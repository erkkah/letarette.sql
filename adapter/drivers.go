package adapter

var drivers = []string{}

func addDriver(driverName string) {
	drivers = append(drivers, driverName)
}

func getLoadedDrivers() []string {
	return drivers
}
