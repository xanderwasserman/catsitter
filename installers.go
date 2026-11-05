// Package catsitter bundles the install scripts so a running Catsitter can
// update itself the same way a manual install does.
package catsitter

import _ "embed"

// InstallSh is the macOS and Linux installer.
//
//go:embed install.sh
var InstallSh []byte

// InstallPs1 is the Windows installer.
//
//go:embed install.ps1
var InstallPs1 []byte
