package verify

// Version is set at build time via -ldflags="-X github.com/tickstem/verify.Version=vX.Y.Z".
// Falls back to "dev" when built without ldflags (local development).
var Version = "dev"
