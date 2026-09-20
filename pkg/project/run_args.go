package project

func pulumiCommandArgs(command string, refresh bool) []string {
	switch command {
	case "diff":
		return []string{"preview"}
	case "refresh":
		return []string{"refresh", "--yes", "--run-program"}
	case "deploy":
		return []string{"up", "--yes", "-f"}
	case "remove":
		args := []string{"destroy", "--yes", "-f"}
		if refresh {
			// Refresh the recorded resources inside the native destroy operation.
			// Do not run the program: it may no longer evaluate after a partial removal.
			args = append(args, "--refresh")
		}
		return args
	default:
		return nil
	}
}
