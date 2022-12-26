package main

type DriverManager struct {
	pathToSteam string
	drivers     map[string]Driver
}

func (d *DriverManager) Push(id string, files []string) ([]string, error) {
	driver := d.drivers[id]
	driver.Push(files)

	return nil, nil
}

type Driver interface {
	Pull([]string)
	Push([]string)
}
