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
	root.AddCommand(NewRunCmd())
	root.AddCommand(NewModeCmd())
	root.AddCommand(NewConfigCmd())
	root.AddCommand(NewKeywordsCmd())
	root.AddCommand(NewCompletionCmd())
	root.AddCommand(NewWhyCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
