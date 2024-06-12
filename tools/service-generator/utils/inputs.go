// Package utils wraps all the utility methods
package utils

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// Config struct contains the user configuration for microservice
type Config struct {
	ServiceName      string
	UseMySQL         bool
	UsePgSQL         bool
	UseKafkaConsumer bool
	UseKafkaProducer bool
	UserName         string
	UserEmail        string
}

// ConfigFromPrompt prompts the user for the configuration
func ConfigFromPrompt(stdin io.Reader, stdout io.Writer) (Config, error) {
	config := Config{}

	prompts := []struct {
		message   string
		processor func(string)
	}{
		{
			message: "Name of the service : ",
			processor: func(value string) {
				config.ServiceName = value
			},
		},
		{
			message: "Do you want to include MySQL [y/N] : ",
			processor: func(value string) {
				config.UseMySQL = strings.ToUpper(value) == "Y"
			},
		},
		{
			message: "Do you want to include PostgreSQL [y/N] : ",
			processor: func(value string) {
				config.UsePgSQL = strings.ToUpper(value) == "Y"
			},
		},
		{
			message: "Do you want to include Kafka Consumer [y/N] : ",
			processor: func(value string) {
				config.UseKafkaConsumer = strings.ToUpper(value) == "Y"
			},
		},
		{
			message: "Do you want to include Kafka Producer [y/N] : ",
			processor: func(value string) {
				config.UseKafkaProducer = strings.ToUpper(value) == "Y"
			},
		},
	}

	consoleReader := bufio.NewReader(stdin)

	for _, p := range prompts {
		fmt.Fprint(stdout, p.message)
		input, err := consoleReader.ReadString('\n')
		if err != nil {
			return config, err
		}
		p.processor(strings.TrimSpace(input))
	}

	parsedName, err := ParseServiceName(config.ServiceName)
	if err != nil {
		return config, err
	}
	config.ServiceName = parsedName

	return config, nil
}

// ConfigFromFlags returns the parsed user inputs from the command line arguments
// which are received as array of strings in the function for better testing.
func ConfigFromFlags(args []string) (Config, error) {
	config := Config{}

	generatorFlags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	generatorFlags.StringVar(&config.ServiceName, "name", "", "Name of the microservice")
	generatorFlags.BoolVar(&config.UseMySQL, "mysql", false, "Include MySQL integration")
	generatorFlags.BoolVar(&config.UsePgSQL, "pgsql", false, "Include PostgreSQL integration")
	generatorFlags.BoolVar(&config.UseKafkaConsumer, "kafkac", false, "Include Kafka consumer")
	generatorFlags.BoolVar(&config.UseKafkaProducer, "kafkap", false, "Include Kafka producer")

	args = args[1:] // slice of arguments without the binary name

	if err := generatorFlags.Parse(args); err != nil {
		return config, err
	}

	parsedName, err := ParseServiceName(config.ServiceName)
	if err != nil {
		return config, err
	}
	config.ServiceName = parsedName

	return config, nil
}
