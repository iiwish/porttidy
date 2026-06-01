//go:build !darwin

package platform

import "fmt"

func GetCWD(pid int32) (string, error) {
	return "", fmt.Errorf("not implemented on this platform")
}

func GetPorts(pid int32) ([]int, error) {
	return nil, fmt.Errorf("not implemented on this platform")
}

func GetAllPorts() (map[int32][]int, error) {
	return nil, fmt.Errorf("not implemented on this platform")
}

func IsProcessAlive(pid int32) bool {
	return false
}

func TerminateProcess(pid int32) error {
	return fmt.Errorf("not implemented on this platform")
}

func KillProcess(pid int32) error {
	return fmt.Errorf("not implemented on this platform")
}
