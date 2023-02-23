# hz.tools/tap

> :warning: Please read [Expectations within this Organization](https://github.com/hztools/.github/tree/main/profile#expectations-within-this-organization) before using it.

## Linux

```
$ sudo setcap cap_net_admin=+ep $(which binary)
```

### Linux specific API

```go
```

## OpenBSD

While this code may compile and run on other BSD flavors, only OpenBSD is
supported. OpenBSD-specific toggles may be added at any point without effort
to keep other BSDs working.

### OpenBSD specific API

```go
// SetDescription will set the network interface description to the provided
// string.
func (i Interface) SetDescription(descr string) error

// AddGroup will add the provided group name to the underlying TAP
// interface.
func (i Interface) AddGroup(name string) error

// RemoveGroup will remove the group name from the TAP interface.
func (i Interface) RemoveGroup(name string) error
```
