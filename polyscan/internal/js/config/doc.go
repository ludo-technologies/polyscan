// Package config holds the settings of the JavaScript/TypeScript analysis.
//
// DefaultConfig returns jscan's settings. polyscan reads no file into them:
// the complexity thresholds and the extra exclude patterns come from
// .polyscan.toml through js.Config, and every other setting keeps its
// default.
package config
