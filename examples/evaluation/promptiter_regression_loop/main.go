//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"flag"
	"log"
)

var (
	configDir = flag.String("config-dir", "configs", "Directory containing regression-loop config files")
	outputDir = flag.String("output-dir", "report", "Directory where optimization_report files are written")
)

func main() {
	flag.Parse()
	if _, err := RunRegressionLoop(RegressionLoopConfig{
		ConfigDir: *configDir,
		OutputDir: *outputDir,
	}); err != nil {
		log.Fatal(err)
	}
}
