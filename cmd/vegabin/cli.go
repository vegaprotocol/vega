package main

import (
	"github.com/spf13/cobra"
)

// Option uses to define the global options.
type Option struct {
	Debug bool
}

type Cli struct {
	Option
	rootCmd   *cobra.Command
//	APIClient client.CommonAPIClient
	padding   int
}

var aboutVega = `  
 __      __  ______    _____            
 \ \    / / |  ____|  / ____|     /\    
  \ \  / /  | |__    | |  __     /  \   
   \ \ \/   |  __|   | | |_ |   / /\ \  
    \ \     | |____  | |__| |  / ____ \ 
     \/     |______|  \_____| /_/    \_\

`


// NewCli creates an instance of 'Cli'.
func NewCli() *Cli {
	return &Cli{
		rootCmd: &cobra.Command{
			Use:   "vega",
			Short: "Smart infrastructure for a better financial system.",
			Long:  aboutVega,
			DisableAutoGenTag: true,
		},
		padding: 3,
	}
}

// Run executes the client program.
func (c *Cli) Run() error {
	return c.rootCmd.Execute()
}

// AddCommand add a sub-command.
func (c *Cli) AddCommand(parent, child Command) {
	child.Init(c)

	parentCmd := parent.Cmd()
	childCmd := child.Cmd()

	// make command error not return command usage and error
	childCmd.SilenceUsage = true
	childCmd.SilenceErrors = true
	childCmd.DisableFlagsInUseLine = true

	childCmd.PreRun = func(cmd *cobra.Command, args []string) {
		c.InitLog()
		//c.NewAPIClient()
	}

	parentCmd.AddCommand(childCmd)
}

// SetFlags sets all global options.
func (c *Cli) SetFlags() *Cli {
//	flags := c.rootCmd.PersistentFlags()
	//flags.StringVarP(&c.Option.host, "host", "H", "unix:///var/run/pouchd.sock", "Specify connecting address of Pouch CLI")
	//flags.BoolVarP(&c.Option.Debug, "debug", "D", false, "Switch client log level to DEBUG mode")
	//flags.StringVar(&c.Option.TLS.Key, "tlskey", "", "Specify key file of TLS")
	//flags.StringVar(&c.Option.TLS.Cert, "tlscert", "", "Specify cert file of TLS")
	//flags.StringVar(&c.Option.TLS.CA, "tlscacert", "", "Specify CA file of TLS")
	//flags.BoolVar(&c.Option.TLS.VerifyRemote, "tlsverify", false, "Use TLS and verify remote")
	return c
}

// InitLog initializes log Level and log format of client.
func (c *Cli) InitLog() {
	if c.Option.Debug {
		//logrus.SetLevel(logrus.DebugLevel)
		//logrus.Infof("start client at debug level")
	}

	//log.InitConsoleLogger(log.DebugLevel)
	//log.InitExitHandler()
	
	//formatter := &logrus.TextFormatter{
	//	FullTimestamp:   true,
	//	TimestampFormat: time.RFC3339Nano,
	//}
	//logrus.SetFormatter(formatter)
}




