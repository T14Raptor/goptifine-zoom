package main

import (
	"fmt"
	"github.com/TheTitanrain/w32"
	"github.com/kindlyfire/go-keylogger"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

func main() {
	processId := getGameProcessId()
	gameId := getGameModule(processId)
	handle, err := w32.OpenProcess(w32.PROCESS_ALL_ACCESS, true, uintptr(processId))
	if err != nil {
		panic(err)
	}

	fmt.Println("Checking if game window is present...")
	gameWindow := w32.FindWindowW(nil, windows.StringToUTF16Ptr("Minecraft"))
	if gameWindow == 0 {
		fmt.Println("Game window not found. Please open minecraft and try again.")
		return
	} else {
		fmt.Println("Game window found! To toggle optifine zoom press C.")
	}

	fovPtr := findAddressFromPointer(handle, gameId+0x03F58580, []uintptr{0x28, 0xFC8, 0x8, 0xC8, 0x170, 0x128, 0x18})

	zoomValue := float32(0.0)
	actualFov := float32(0.0)
	toggled := false

	kl := keylogger.NewKeylogger()
	for {
		key := kl.GetKey()
		if key.Rune == 'c' {
			if !toggled {
				zoomValue = 10.0
				readProcessMemory(handle, fovPtr, uintptr(unsafe.Pointer(&actualFov)), unsafe.Sizeof(actualFov))
			} else {
				zoomValue = actualFov
			}

			w32.WriteProcessMemory(handle, fovPtr, uintptr(unsafe.Pointer(&zoomValue)), unsafe.Sizeof(zoomValue))
			toggled = !toggled
		}
	}
}

func getGameProcessId() uint32 {
	var processID uint32
	snapshotHandle, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	defer syscall.Close(snapshotHandle)
	if err == nil {
		var processEntry syscall.ProcessEntry32
		processEntry.Size = uint32(unsafe.Sizeof(processEntry))
		if syscall.Process32First(snapshotHandle, &processEntry) == nil {
			for {
				if windows.UTF16ToString(processEntry.ExeFile[:]) == "Minecraft.Windows.exe" {
					processID = processEntry.ProcessID
					break
				}

				if err := syscall.Process32Next(snapshotHandle, &processEntry); err != nil {
					break
				}
			}
		}
	} else {
		panic(err)
	}

	return processID
}

func findAddressFromPointer(proc w32.HANDLE, providedPtr uintptr, providedOffsets []uintptr) uintptr {
	address := providedPtr
	for _, offset := range providedOffsets {
		readProcessMemory(proc, address, uintptr(unsafe.Pointer(&address)), unsafe.Sizeof(address))
		address += offset
	}

	return address
}

var (
	modkernel32           = syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMemory = modkernel32.NewProc("ReadProcessMemory")
)

func readProcessMemory(hProcess w32.HANDLE, lpBaseAddress, lpBuffer, nSize uintptr) (lpNumberOfBytesRead int, ok bool) {
	var nBytesRead int
	ret, _, _ := procReadProcessMemory.Call(
		uintptr(hProcess),
		lpBaseAddress,
		lpBuffer,
		nSize,
		uintptr(unsafe.Pointer(&nBytesRead)),
	)

	return nBytesRead, ret != 0
}

func getGameModule(processID uint32) uintptr {
	var gameModuleAddress uintptr = 0
	snapshotHandle := w32.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPMODULE|syscall.TH32CS_SNAPMODULE32, processID)
	defer w32.CloseHandle(snapshotHandle)

	if snapshotHandle != w32.ERROR_INVALID_HANDLE {
		var modEntry w32.MODULEENTRY32
		modEntry.Size = uint32(unsafe.Sizeof(modEntry))

		if w32.Module32First(snapshotHandle, &modEntry) {
			for {
				if windows.UTF16ToString(modEntry.SzModule[:]) == "Minecraft.Windows.exe" {
					fmt.Println("found game id")
					gameModuleAddress = uintptr(unsafe.Pointer(modEntry.ModBaseAddr))
					break
				}

				if w32.Module32Next(snapshotHandle, &modEntry) {
					break
				}
			}
		}
	}

	return gameModuleAddress
}
