# fluent.go &nbsp; [![](https://pkg.go.dev/badge/github.com/lus/fluent.go/fluent.svg)](https://pkg.go.dev/github.com/lus/fluent.go/fluent) [![](https://github.com/lus/fluent.go/actions/workflows/test.yml/badge.svg)](https://github.com/lus/fluent.go/actions/workflows/test.yml) [![](https://bors.tech/images/badge_small.svg)](https://app.bors.tech/repositories/41488)



`fluent.go` is a pure Go implementation of [Project Fluent](https://projectfluent.org)

## Important notice

Even though the primary goal of this project is to exactly replicate the behaviour of the official Project Fluent
implementations,
some key features are **not yet implemented**.

These are:

* date & time handling
* locale-based number formatting (replica of `Intl.NumberFormat`)
* the `NUMBER` and `DATETIME` builtin functions

These will, of course, be implemented as soon as possible.

## FTL Syntax

If you are not familiar with Project Fluent and it's syntax, head over to the project's
[Homepage](https://projectfluent.org) and read the well-written [Syntax Guide](https://projectfluent.org/fluent/guide).

## Quickstart

### Adding the dependency

First of all you need to add the `fluent` package to your module:

```shell
go get -u github.com/lus/fluent.go/fluent
```

### Creating a `Resource`

`Resource`s simply extract and hold a set of `Message`s and `Term`s out of a FTL source:

```go
// You need a string that represents a FTL source.
ftl := "greeting = Hello, { $subject }!"

// Now we create a resource using this source.
// An important note here is that 'errs' are the errors the parser stumbled upon during parsing.
// It is highly recommended handling these, but receiving errors here does not necessarily mean
// that no messages or terms could be loaded at all. Some may have succeeded. 
resource, errs := fluent.NewResource(ftl)
```

### Creating a `Bundle` and adding resources

`Bundle`s are assembled using one or multiple `Resource`s and provide the main API to actually localize messages:

```go
ftl := "greeting = Hello, { $subject }!"
resource, errs := fluent.NewResource(ftl)
// Error handling is recommended!

// The next and final entity you will need is the Fluent bundle.
// It loads resources and actually formats the messages.
// The fluent.NewBundle method takes at least one language tag (EN in this case).
// The first one represents the primary language tag, the other ones are fallbacks.
// These are used to format numbers and dates.
bundle := fluent.NewBundle(language.EN)

// bundle.AddResource loads all messages and terms present in a resource into the bundle.
// If a message or term is already present in the bundle, an error is raised and the entry is skipped.
// These errors are returned afterwards.
errs = bundle.AddResource(resource)

// If you want to override existing entries instead, use bundle.AddResourceOverriding.
// It does not return any errors.
bundle.AddResourceOverriding(resource)
```

### Formatting messages

Now that we have a bundle with a message named `greeting`, we can format it with our context:

```go
ftl := "greeting = Hello, { $subject }!"
resource, errs := fluent.NewResource(ftl)
// Error handling is recommended!

bundle := fluent.NewBundle(language.EN)
bundle.AddResourceOverriding(resource)

// Now we can use the bundle.FormatMessage method to actually format our greeting.
//
// Firstly, this method returns the formatted string representation of the message.
//
// The second returned value are errors that occurred while resolving certain parts of the message.
// An example for that would be if a variable or function does not exist.
// The final string would contain something like '{$varName}' instead of the actual value in that case.
//
// The third return value is an error that indicates that the whole formatting process failed.
// This would be the case if there is no message with the given key.
message, errs, fatalErr := bundle.FormatMessage("greeting", fluent.WithVariable("subject", "world"))
// Error handling is recommended!

fmt.Println(message)
// -> Hello, world!
```

### Message attributes

A message may declare attributes beside — or instead of — a value of its own,
which is how one id can carry several related strings:

```ftl
login-code =
    .subject = Your code
    .body = Your code is { $code }.
```

`FormatMessage` formats a message's own value and reports an error for a message
like the one above, which has none. Use `FormatMessageAttribute` to format one of
its attributes:

```go
subject, errs, fatalErr := bundle.FormatMessageAttribute("login-code", "subject")

body, errs, fatalErr := bundle.FormatMessageAttribute("login-code", "body",
	fluent.WithVariable("code", "123456"))
```

`HasMessage` and `HasMessageAttribute` answer whether a bundle holds either,
without formatting it.

### Dates

`DATETIME` is available to every bundle without being registered, and formats a
`time.Time` (or an RFC 3339 string, which is what a timestamp that has been
through JSON looks like):

```ftl
suspended = Your account is suspended until { DATETIME($until, dateStyle: "short", timeStyle: "short", timeZoneName: "short") }.
```

```go
message, errs, fatalErr := bundle.FormatMessage("suspended",
	fluent.WithVariable("until", time.Now().Add(72*time.Hour)))
```

**It formats in a locale-neutral way, and that is deliberate.** Fluent models
`DATETIME` on ECMA-402's `Intl.DateTimeFormat`, whose whole subject is writing a
date the way a given locale writes one. Go ships no CLDR date data, so this
implementation cannot do that — and writing "September" for every locale would be
worse than not offering it, because it would look localized. So the output is
ISO 8601 order and numeric throughout, which every locale can read:

| Options | Output |
| --- | --- |
| *(none)* | `2026-09-01 12:30 UTC` |
| `dateStyle: "short"` | `2026-09-01` |
| `timeStyle: "short"` | `12:30` |
| `timeStyle: "long"` | `12:30:45 UTC` |
| `timeZone: "America/Sao_Paulo", timeStyle: "short"` | `09:30` |

The options are named as ECMA-402 names them, so a catalog written against a
fuller implementation keeps working. `dateStyle` and `timeStyle` take
`full`/`long`/`medium`/`short`; the component options (`year`, `month`, `day`,
`hour`, `minute`, `second`) take `numeric` and `2-digit`. Anything that can only
mean "write a name" — `month: "long"` and friends — records an error and falls
back to digits rather than inventing English.

**The default time zone is UTC**, not the machine's: a formatted date with no
zone in it means something different to every reader, and a server rarely knows
which zone its reader is in. `timeZone` takes an IANA name, resolved through
`time.LoadLocation`, so anything but `UTC` needs tzdata present (import
`time/tzdata` to embed it).

If you do have locale-aware formatting available, pass your own `DATETIME`
through `fluent.WithFunction` and it takes precedence over this one.

### Further information

For further information about how to use the API head over to the
[API reference](https://pkg.go.dev/github.com/lus/fluent.go/fluent).

## Bugs & Improvements

Feel free to open an issue or pull request if you encounter any bugs or if you want to suggest something.

You may also join my [Discord server](https://go.lus.pm/discord) if you prefer this way of communication.
