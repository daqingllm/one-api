package common

import (
	"flag"
	"fmt"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/logger"
	"log"
	"os"
	"path/filepath"
)

func printHelp() {
	fmt.Println("AiHubMix " + Version + " - All in one API service for OpenAI API.")
	fmt.Println("Copyright (C) 2023 AiHubMix. All rights reserved.")
	fmt.Println("GitHub: https://github.com/euansu/AIHubMix")
	fmt.Println("Usage: one-api [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

func init() {
	flag.Parse()

	if *env.PrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *env.PrintHelp {
		printHelp()
		os.Exit(0)
	}

	if os.Getenv("SESSION_SECRET") != "" {
		if os.Getenv("SESSION_SECRET") == "random_string" {
			logger.SysError("SESSION_SECRET is set to an example value, please change it to a random string.")
		} else {
			config.SessionSecret = os.Getenv("SESSION_SECRET")
		}
	}
	if *env.LogDir != "" {
		var err error
		*env.LogDir, err = filepath.Abs(*env.LogDir)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := os.Stat(*env.LogDir); os.IsNotExist(err) {
			err = os.Mkdir(*env.LogDir, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
		logger.LogDir = *env.LogDir
	}
}
