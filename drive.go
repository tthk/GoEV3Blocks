package GoEV3Blocks

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tthk/GoEV3/Motor"
	"github.com/tthk/GoEV3/Sensors"
)

type Drive struct {
	leftMotor, rightMotor     Motor.OutPort
	brakeModeValid, brakeMode bool
	moveChannel               chan bool
	gyro                      *Sensors.GyroSensor
}

type MoveOp int

// Move operation constants
const (
	MOVE_OFF MoveOp = iota // Using Go's iota to give incremental constant values
	MOVE_ON
	MOVE_ON_GYRO
	MOVE_ON_SECONDS
	MOVE_ON_DEGREES
	MOVE_ON_ROTATIONS
)

// String() interface to allow printing of Move_Op constant
func (op MoveOp) String() string {
	switch {
	case op == MOVE_OFF:
		return "MOVE_OFF"
	case op == MOVE_ON:
		return "MOVE_ON"
	case op == MOVE_ON_GYRO:
		return "MOVE_ON_GYRO"
	case op == MOVE_ON_SECONDS:
		return "MOVE_ON_SECONDS"
	case op == MOVE_ON_DEGREES:
		return "MOVE_ON_DEGREES"
	case op == MOVE_ON_ROTATIONS:
		return "MOVE_ON_ROTATIONS"
	default:
		return fmt.Sprintf("MOVE_?: %v", op)
	}
}

// Configures the left and right motors to their ports and some states
func (d *Drive) Configure(leftMotor, rightMotor Motor.OutPort, gyro *Sensors.GyroSensor, regulationMode bool) {
	d.leftMotor = leftMotor
	d.rightMotor = rightMotor
	d.gyro = gyro
	if regulationMode {
		Motor.EnableRegulationMode(leftMotor)
		Motor.EnableRegulationMode(rightMotor)
	} else {
		Motor.DisableRegulationMode(leftMotor)
		Motor.DisableRegulationMode(rightMotor)
	}
}

// Steering power curve function for left wheel based in percent [-100..100] of turn.
// To get right wheel's power curve, pass in negative percent
// Returns percent [-100..100] of power to apply to wheel
func steeringPowerCurve(percent float32) float32 {
	if percent < 0 {
		return (2*percent + 100)
	} else {
		return 100
	}
}

// Intelligently set brakeMode for motors in that this function won't set
// motor to mode it's already at.  State of brakeMode is saved in moveConfig
func (d *Drive) setBrakeMode(brakeMode bool) {
	// Set brake mode on motors
	if d.brakeModeValid != true {
		d.brakeModeValid = true
		d.brakeMode = !brakeMode // This causes the next section to run
	}
	if brakeMode != d.brakeMode {
		// We need to actually set the motor's brake mode
		if brakeMode == true {
			// Hit the brakes
			Motor.EnableBrakeMode(d.leftMotor)
			Motor.EnableBrakeMode(d.rightMotor)
		} else {
			// Coast
			Motor.DisableBrakeMode(d.leftMotor)
			Motor.DisableBrakeMode(d.rightMotor)
		}
		d.brakeMode = brakeMode
	} else {
		// Motor's brake mode already set to what we want, don't do anything
	}
}

func (d *Drive) MoveSteering(op MoveOp, args ...interface{}) error {
	var err error     // Start with no err
	argc := len(args) // How many arguments

	//fmt.Printf("\n%s, %#v\n", op, args)
	if argc == 0 {
		return errors.New("No op specified")
	}
	switch op {
	case MOVE_OFF: // brake bool
		if argc != 1 {
			return errors.New("Expected 1 argument, got " + strconv.Itoa(argc))
		}

		d.setBrakeMode(args[0].(bool))

		if d.moveChannel != nil {
			// We let the moveChannel listener shutdown motor
			close(d.moveChannel)
			d.moveChannel = nil
			time.Sleep(250 * time.Millisecond) // Yield so goroutine to stop may run
		} else {
			// Stop motors
			fmt.Println("Stopping motors!")
			Motor.Stop(d.leftMotor)
			Motor.Stop(d.rightMotor)
		}

	case MOVE_ON: // steering float64, power int
		if argc != 2 {
			return errors.New("Expected 2 arguments, got " + strconv.Itoa(argc))
		}
		steering := float32(args[0].(float64)) // Percent steering
		power := int16(args[1].(int))

		leftPower := float32(power) * steeringPowerCurve(steering) / 100
		rightPower := float32(power) * steeringPowerCurve(-steering) / 100

		Motor.Run(d.leftMotor, int16(leftPower))
		Motor.Run(d.rightMotor, int16(rightPower))

	case MOVE_ON_GYRO: // angle int, power int
		if d.moveChannel == nil {
			d.moveChannel = make(chan bool) // Control channel for move
		}
		if argc != 2 {
			return errors.New("Expected 2 arguments, got " + strconv.Itoa(argc))
		}
		angle := args[0].(int16) // Angle to keep
		power := int16(args[1].(int))

		// Ramp up
		Motor.Run(d.leftMotor, power/2)
		Motor.Run(d.rightMotor, power/2)
		moveChannel := d.moveChannel // So we have handle on moveChannel even after it's removed and nil via closure
		go func(angle, power int16) {
			var rightPower, leftPower int16
			sign := ""
			for {
				select {
				case <-moveChannel:
					fmt.Println("moveChannel STOP")
					return
				default:
					gyroValue := d.gyro.ReadAngle()
					if gyroValue < angle {
						// Adjust by pulling to the right
						leftPower = int16(float32(power) * float32(1.1))
						rightPower = power
						sign = "<"
					} else if gyroValue > angle {
						// Adjust by pulling to the left
						leftPower = power
						rightPower = int16(float32(power) * float32(1.1))
						sign = ">"
					} else {
						leftPower = power
						rightPower = power
						sign = "="
					}
					Motor.Run(d.leftMotor, leftPower)
					Motor.Run(d.rightMotor, rightPower)

					fmt.Printf("(%#v%s%#v,%#v,%#v)", gyroValue, sign, angle, leftPower, rightPower)

				}
			}

		}(angle, power)

	case MOVE_ON_DEGREES: // steering float64, power int, degrees int, brake bool
		if argc != 4 {
			return errors.New("Expected 4 arguments, got " + strconv.Itoa(argc))
		}
		steering := float32(args[0].(float64)) // Percent steering
		power := int16(args[1].(int))
		degrees := int32(args[2].(int)) // Degrees to rotate wheel
		brake := args[3].(bool)         // Percent steering

		d.setBrakeMode(brake)

		leftPower := float32(power) * steeringPowerCurve(steering) / 100
		rightPower := float32(power) * steeringPowerCurve(-steering) / 100

		Motor.RunToRelPos(d.leftMotor, int16(leftPower), degrees)
		Motor.RunToRelPos(d.rightMotor, int16(rightPower), degrees)

		// Wait until we are reach requested degrees
		for {
			if (Motor.ReadState(d.leftMotor) == "") || (Motor.ReadState(d.rightMotor) == "") {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

	case MOVE_ON_ROTATIONS: // steering float64, power int, rotations int, brake bool
		if argc != 4 {
			return errors.New("Expected 4 arguments, got " + strconv.Itoa(argc))
		}
		degrees := args[2].(int) * 360.0 // Rotations to rotate wheel
		d.MoveSteering(MOVE_ON_DEGREES, args[0], args[1], degrees, args[3])

	case MOVE_ON_SECONDS: // steering float64, power int, seconds float64, brake bool
		if argc != 4 {
			return errors.New("Expected 4 arguments, got " + strconv.Itoa(argc))
		}
		steering := float32(args[0].(float64)) // Percent steering
		power := int16(args[1].(int))
		ms := int32(args[2].(float64) * 1000) // Milliseconds to rotate wheel
		brake := args[3].(bool)               // Percent steering

		d.setBrakeMode(brake)

		leftPower := float32(power) * steeringPowerCurve(steering) / 100
		rightPower := float32(power) * steeringPowerCurve(-steering) / 100

		Motor.RunTimed(d.leftMotor, int16(leftPower), ms)
		Motor.RunTimed(d.rightMotor, int16(rightPower), ms)

		// Wait until we are reach requested time
		for {
			if (Motor.ReadState(d.leftMotor) == "") || (Motor.ReadState(d.rightMotor) == "") {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

	default:
		return errors.New("Unknown op: " + strconv.Itoa(int(op)))

	}
	return err
}
