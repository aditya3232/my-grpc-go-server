package application

import (
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc/codes"
)

type ResiliencyService struct{}

func (r *ResiliencyService) GenerateResiliency(minDelaySecond int32, maxDelaySecond int32, statusCodes []uint32) (string, uint32) {
	if len(statusCodes) == 0 {
		return "", uint32(codes.InvalidArgument) // atau default aman lainnya
	}

	min, max := int(minDelaySecond), int(maxDelaySecond)
	if max < min {
		min, max = max, min // atau return error, tergantung kebutuhan
	}

	var delay int
	if max == min {
		delay = min
	} else {
		delay = min + rand.Intn(max-min+1) // inklusif [min, max]
	}

	time.Sleep(time.Duration(delay) * time.Second)

	idx := rand.Intn(len(statusCodes))
	str := fmt.Sprintf("The time now is %v, execution delayed for %v seconds \n", time.Now().Format("15:04:05.000"), delay)

	return str, statusCodes[idx]
}
