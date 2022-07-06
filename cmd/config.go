package cmd

import "github.com/spf13/pflag"

// Config ...
type Config struct {
	Workers int
}

// Gets configuration from CLI flags.
func getConfig(flags *pflag.FlagSet) (Config, error) {
	w, err := flags.GetInt("workers")
	if err != nil {
		return Config{}, err
	}
	return Config{
		Workers: w,
	}, nil
}
