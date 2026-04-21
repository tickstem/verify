# tickstem/verify

[![Go Reference](https://pkg.go.dev/badge/github.com/tickstem/verify.svg)](https://pkg.go.dev/github.com/tickstem/verify)
[![Go Report Card](https://goreportcard.com/badge/github.com/tickstem/verify)](https://goreportcard.com/report/github.com/tickstem/verify)
[![codecov](https://codecov.io/gh/tickstem/verify/badge.svg)](https://codecov.io/gh/tickstem/verify)

Go SDK for [Tickstem](https://tickstem.dev) — email verification for production apps.

Checks syntax, MX records, disposable domains, and role-based prefixes before an address touches your database.

## Install

```bash
go get github.com/tickstem/verify
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/tickstem/verify"
)

func main() {
    client := verify.New(os.Getenv("TICKSTEM_API_KEY"))

    result, err := client.Verify(context.Background(), "user@example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Valid)      // true if safe to store
    fmt.Println(result.Disposable) // true if throwaway address
    fmt.Println(result.RoleBased)  // true if generic inbox (admin@, info@, ...)
    fmt.Println(result.Reason)     // explanation when Valid is false
}
```

Get your API key at [app.tickstem.dev](https://app.tickstem.dev).

## Usage

### Create a client

```go
// Minimal — uses https://api.tickstem.dev/v1
client := verify.New(os.Getenv("TICKSTEM_API_KEY"))

// With options
client := verify.New(apiKey,
    verify.WithBaseURL("http://localhost:8080/v1"),
)
```

### Verify an email

```go
result, err := client.Verify(ctx, "user@example.com")
if err != nil {
    if verify.IsQuotaExceeded(err) {
        // monthly quota hit — upgrade at app.tickstem.dev/dashboard/billing
    }
    log.Fatal(err)
}

if !result.Valid {
    return fmt.Errorf("email not accepted: %s", result.Reason)
}

if result.RoleBased {
    // warn but don't block — some teams share an inbox
}
```

### What gets checked

| Check | What it catches |
|-------|----------------|
| Syntax | Malformed addresses (`user@@example`, missing TLD) |
| MX lookup | Domains with no mail server (`gmial.com`, abandoned domains) |
| Disposable | Throwaway services (Mailinator, Guerrilla Mail, 200+ others) |
| Role-based | Generic inboxes (`admin@`, `info@`, `noreply@`, `support@`, ...) |

No SMTP probing — the recipient mail server is never contacted.

### Retrieve history

```go
page, err := client.ListHistory(ctx, verify.ListHistoryParams{
    Limit:  50,
    Offset: 0,
})
for _, v := range page.Verifications {
    fmt.Printf("%s  valid=%v  disposable=%v\n", v.Email, v.Valid, v.Disposable)
}
```

## Error handling

```go
result, err := client.Verify(ctx, email)
if err != nil {
    if verify.IsUnauthorized(err) {
        // invalid or revoked API key
    }
    if verify.IsQuotaExceeded(err) {
        // monthly verification limit reached
    }
    var apiErr *verify.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode, apiErr.Message)
    }
}
```

## Pricing

| Plan     | Verifications/month | Price  |
|----------|---------------------|--------|
| Free     | 500                 | $0     |
| Starter  | 5,000               | $12/mo |
| Pro      | 50,000              | $29/mo |
| Business | 500,000             | $79/mo |

[View full pricing →](https://tickstem.dev/#pricing)

## License

MIT
