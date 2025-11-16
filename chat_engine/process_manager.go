package chat_engine

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ProcessInfo struct {
	PID            int       `json:"pid"`
	Command        string    `json:"command"`
	StartTime      time.Time `json:"start_time"`
	ConversationID string    `json:"conversation_id,omitempty"`
}

type ProcessManager struct {
	processes map[int]*ProcessInfo
	mutex     sync.RWMutex
}

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		processes: make(map[int]*ProcessInfo),
	}

	// Cleanup on exit
	go pm.setupCleanup()

	return pm
}

func (pm *ProcessManager) setupCleanup() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Kill all processes on exit
	pm.KillAll()
	os.Exit(0)
}

func (pm *ProcessManager) StartProcess(command string, conversationID string) (*ProcessInfo, error) {
	cmd := exec.Command("bash", "-c", command)

	// Set process group so we can kill child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	pid := cmd.Process.Pid
	info := &ProcessInfo{
		PID:            pid,
		Command:        command,
		StartTime:      time.Now(),
		ConversationID: conversationID,
	}

	pm.mutex.Lock()
	pm.processes[pid] = info
	pm.mutex.Unlock()

	// Monitor process in background
	go func() {
		cmd.Wait()
		pm.mutex.Lock()
		delete(pm.processes, pid)
		pm.mutex.Unlock()
		log.Printf("Process %d finished: %s", pid, command)
	}()

	log.Printf("Started background process PID: %d, Command: %s", pid, command)
	return info, nil
}

func (pm *ProcessManager) ListProcesses() []*ProcessInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	processes := make([]*ProcessInfo, 0, len(pm.processes))
	for _, info := range pm.processes {
		// Check if process is still running
		process, err := os.FindProcess(info.PID)
		if err == nil {
			err = process.Signal(syscall.Signal(0)) // Signal 0 checks if process exists
			if err == nil {
				processes = append(processes, info)
			} else {
				// Process is dead, remove it
				delete(pm.processes, info.PID)
			}
		}
	}

	return processes
}

func (pm *ProcessManager) KillProcess(pid int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	info, exists := pm.processes[pid]
	if !exists {
		return fmt.Errorf("process %d not found", pid)
	}

	// Kill the process group to kill all children (negative PID kills the group)
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil {
		// Try killing just the process
		process, err2 := os.FindProcess(pid)
		if err2 != nil {
			return fmt.Errorf("failed to find process: %w", err2)
		}
		err = process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	delete(pm.processes, pid)
	log.Printf("Killed process %d (and its process group): %s", pid, info.Command)
	return nil
}

func (pm *ProcessManager) KillAll() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for pid, info := range pm.processes {
		process, err := os.FindProcess(pid)
		if err == nil {
			syscall.Kill(-pid, syscall.SIGTERM)
			process.Kill()
			log.Printf("Killed process %d: %s", pid, info.Command)
		}
		delete(pm.processes, pid)
	}
}

func (pm *ProcessManager) KillByConversation(conversationID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for pid, info := range pm.processes {
		if info.ConversationID == conversationID {
			process, err := os.FindProcess(pid)
			if err == nil {
				syscall.Kill(-pid, syscall.SIGTERM)
				process.Kill()
				log.Printf("Killed process %d from conversation %s: %s", pid, conversationID, info.Command)
			}
			delete(pm.processes, pid)
		}
	}
}

