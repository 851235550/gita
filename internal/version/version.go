// Package version 提供编译时注入的版本信息。
// 通过 -ldflags 在编译时注入，默认值为 "dev"。
package version

// Version 版本号，由 git describe --tags 生成。
var Version = "dev"

// Commit 构建时的 Git commit SHA（短格式）。
var Commit = "unknown"

// BuildTime 构建时间（UTC）。
var BuildTime = "unknown"
