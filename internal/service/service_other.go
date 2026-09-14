//go:build !darwin && !linux

package service

func Install(bin, root string) (Info, error) { return Info{}, ErrUnsupported }
func Uninstall() error                       { return ErrUnsupported }
func Status() (Info, error)                  { return Info{}, ErrUnsupported }
