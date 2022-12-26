package main

type DriverManager struct {
	drivers map[uint64]Driver
}

func (d *DriverManager) GetId(name string) uint64 {
	return 0
}

func (d *DriverManager) Push(id uint64) ([]string, error) {
	driver := d.getDriver(id)
	driver.PrePush()
	driver.Push()
	driver.PostPush()

	return nil, nil
}

func (d *DriverManager) getDriver(id uint64) Driver {
	return nil
}

type Driver interface {
	PrePull()
	Pull()
	PostPull()

	PrePush()
	Push()
	PostPush()
}
