package main

import "os"

func main() {
	root := NewRootCmd()
	root.AddCommand(NewVersionCmd())
	root.AddCommand(NewInitCmd())
	root.AddCommand(NewDetectCmd())
	root.AddCommand(NewHarnessCmd())
	root.AddCommand(NewDoctorCmd())
	root.AddCommand(NewSetupCmd())
	root.AddCommand(NewInstallHookCmd())
	root.AddCommand(NewRunCmd())
	root.AddCommand(NewModeCmd())
	root.AddCommand(NewConfigCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
