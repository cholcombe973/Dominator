package herd

import (
	"github.com/Symantec/Dominator/dom/lib"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"syscall"
	"time"
)

// Returns (idle, missing), idle=true if no update needs to be performed.
func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) (
	bool, bool) {
	sub.herd.computeSemaphore <- struct{}{}
	defer func() { <-sub.herd.computeSemaphore }()
	requiredImage := sub.herd.getImageNoError(sub.getRequiredImageName())
	request.ImageName = sub.getRequiredImageName()
	request.Triggers = requiredImage.Triggers
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	subObj := lib.Sub{
		Hostname:       sub.mdb.Hostname,
		FileSystem:     sub.fileSystem,
		ComputedInodes: sub.computedInodes,
		ObjectCache:    sub.objectCache}
	if lib.BuildUpdateRequest(subObj, requiredImage, request, false,
		sub.herd.logger) {
		return false, true
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	sub.lastComputeUpdateCpuDuration = time.Duration(
		rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	computeCpuTimeDistribution.Add(sub.lastComputeUpdateCpuDuration)
	if len(request.FilesToCopyToCache) > 0 ||
		len(request.InodesToMake) > 0 ||
		len(request.HardlinksToMake) > 0 ||
		len(request.PathsToDelete) > 0 ||
		len(request.DirectoriesToMake) > 0 ||
		len(request.InodesToChange) > 0 ||
		sub.lastSuccessfulImageName != sub.getRequiredImageName() {
		sub.herd.logger.Printf(
			"buildUpdateRequest(%s) took: %s user CPU time\n",
			sub, sub.lastComputeUpdateCpuDuration)
		return false, false
	}
	return true, false
}
