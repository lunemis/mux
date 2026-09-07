package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/lunemis/mux/config"
	"github.com/lunemis/mux/theme"
	"github.com/lunemis/mux/tmux"
	"github.com/lunemis/mux/ui"
)

var version = "dev"

func main() {
	settings, configErr := config.Load()
	themeName := configuredThemeNameFrom(settings)
	configError := func(cmd *cobra.Command) error {
		themeOverride := cmd.Flags().Changed("theme") || os.Getenv("MUX_THEME") != ""
		if configErr != nil && !themeOverride {
			return configErr
		}
		return nil
	}

	rootCmd := &cobra.Command{
		Use:     "mux",
		Short:   "TUI tmux session manager",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configError(cmd); err != nil {
				return err
			}
			keyMap, err := keyMapFromSettings(settings)
			if err != nil {
				return err
			}
			return runTUI(cmd, args, keyMap)
		},
		// Suppress cobra's default completion and help subcommands
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	rootCmd.SetVersionTemplate("mux {{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&themeName, "theme", themeName,
		fmt.Sprintf("Color theme (%s)", joinWith(theme.Names(), ", ")))

	popupCmd := &cobra.Command{
		Use:   "popup",
		Short: "Open mux as a tmux popup overlay",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configError(cmd); err != nil {
				return err
			}
			if _, err := keyMapFromSettings(settings); err != nil {
				return err
			}
			if _, err := theme.Get(themeName); err != nil {
				return err
			}
			return tmux.OpenPopup("--theme", themeName)
		},
	}

	setupKeybindCmd := &cobra.Command{
		Use:   "setup-keybind [key]",
		Short: fmt.Sprintf("Add popup keybinding to tmux config (default: %s)", tmux.DefaultBindKey),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := tmux.DefaultBindKey
			if len(args) > 0 {
				key = args[0]
			}
			return tmux.SetupKeybind(key)
		},
	}

	rootCmd.AddCommand(popupCmd, setupKeybindCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func joinWith(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

func configuredThemeName() (string, error) {
	if name := os.Getenv("MUX_THEME"); name != "" {
		return name, nil
	}
	settings, err := config.Load()
	if err != nil {
		return "", err
	}
	return configuredThemeNameFrom(settings), nil
}

func configuredThemeNameFrom(settings config.Config) string {
	if name := os.Getenv("MUX_THEME"); name != "" {
		return name
	}
	if settings.Theme != "" {
		return settings.Theme
	}
	return "default"
}

func configuredKeyMap() (ui.KeyMap, error) {
	settings, err := config.Load()
	if err != nil {
		return ui.KeyMap{}, err
	}
	return keyMapFromSettings(settings)
}

func keyMapFromSettings(settings config.Config) (ui.KeyMap, error) {
	keyMap, err := ui.NewKeyMap(settings.Keybindings)
	if err == nil {
		return keyMap, nil
	}
	path, pathErr := config.Path()
	if pathErr != nil {
		return ui.KeyMap{}, err
	}
	return ui.KeyMap{}, fmt.Errorf("invalid keybindings in %s: %w", path, err)
}

func configureTheme(name string) error {
	selected, err := theme.Get(name)
	if err != nil {
		return err
	}
	ui.UseTheme(selected)
	return nil
}

func runTUI(cmd *cobra.Command, args []string, keyMap ui.KeyMap) error {
	name, err := cmd.Flags().GetString("theme")
	if err != nil {
		return err
	}
	if err := configureTheme(name); err != nil {
		return err
	}

	p := tea.NewProgram(ui.NewModelWithKeyMap(keyMap), tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if m, ok := result.(ui.Model); ok {
		if name := m.AttachName(); name != "" {
			if err := ui.AttachToSession(name, m.AttachWindowIndex(), m.AttachPaneIndex()); err != nil {
				return fmt.Errorf("failed to attach: %w", err)
			}
		}
	}
	return nil
}
