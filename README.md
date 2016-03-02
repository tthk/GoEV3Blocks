# GoEV3Drive
Implements Drive structure using two GoEV3 motors, gyro, etc...
Requires github.com/ttkt/GoEV3 package, which is based on https://github.com/ldmberman/GoEV3

Usage:

	import "github.com/ttkt/GoEV3Blocks"
	
	var drive GoEV3Blocks.Drive
	
	func main() {
		// Configure the drive GoEV3Blocks.Drive structure
		// gyro parameter may be nil if MOVE_ON_GYRO is never used
		gyro := Sensors.FindGyroSensor(Sensors.InPort4)
		drive.Configure(Motor.OutPortB, Motor.OutPortC, gyro)
		
		// Stop wheels
		if drive.MoveSteering(GoEV3Blocks.MOVE_OFF, true) != nil {
			log.Println(err)
		}
	
		// Move straight ahead, 100 power, for 360 degrees wheel rotation
		if drive.MoveSteering(GoEV3Blocks.MOVE_ON_DEGREES, 0.0, 100, 360, true) != nil {
			log.Println(err)
		}
		
		// Move straight ahead, 100 power, for 2 full wheel rotations
		if drive.MoveSteering(GoEV3Blocks.MOVE_ON_ROTATIONS, 0.0, 100, 2.0, true) != nil {
			log.Println(err)
		}
		
		// Move straight ahead, 100 power, for 5 seconds degrees wheel rotation
		if drive.MoveSteering(GoEV3Blocks.MOVE_ON_SECONDS, 0.0, 100, 5.0, true) != nil {
			log.Println(err)
		}
		
		// Move straight on gyro, keeping at 90 degrees, power 300
		if drive.MoveSteering(GoEV3Blocks.MOVE_ON_GYRO, 90, 300) != nil {
			log.Println(err)
		}

	}

Notes:
	This is very preliminary work in progress.
	