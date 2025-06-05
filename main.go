package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiskInfo represents information about a disk partition
type DiskInfo struct {
	Filesystem string
	Size       int64
	Used       int64
	Available  int64
	UsePercent int
	MountPoint string
}

// Model represents the application state
type Model struct {
	diskInfos []DiskInfo
	err       error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return getDiskInfoCmd
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	if len(m.diskInfos) == 0 {
		return "Loading disk information..."
	}

	var sb strings.Builder

	// Define styles with modern colors
	fsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))      // Light blue
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94E2D5"))    // Cyan
	mountStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))   // Green
	usedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))    // Red
	availStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))   // Green
	percentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387")) // Orange

	// Write disk info with graph
	for _, di := range m.diskInfos {
		// Skip pseudo filesystems
		if strings.HasPrefix(di.Filesystem, "/dev") || strings.HasPrefix(di.Filesystem, "/") {
			// Format sizes in human-readable format
			size := formatSize(di.Size)
			used := formatSize(di.Used)
			avail := formatSize(di.Available)

			// Create usage bar
			const barWidth = 30
			usedChars := int(float64(di.UsePercent) / 100.0 * float64(barWidth))
			availChars := barWidth - usedChars

			usedBar := usedStyle.Render(strings.Repeat("█", usedChars))
			availBar := availStyle.Render(strings.Repeat("░", availChars))

			// Create a device info block
			deviceInfo := fmt.Sprintf("%s\n", fsStyle.Render(di.Filesystem))
			deviceInfo += fmt.Sprintf("├─ Size: %s\n", sizeStyle.Render(size))
			deviceInfo += fmt.Sprintf("├─ Used: %s\n", usedStyle.Render(used))
			deviceInfo += fmt.Sprintf("├─ Available: %s\n", availStyle.Render(avail))
			deviceInfo += fmt.Sprintf("├─ Usage: %s\n", percentStyle.Render(fmt.Sprintf("%d%%", di.UsePercent)))
			deviceInfo += fmt.Sprintf("├─ Mounted on: %s\n", mountStyle.Render(di.MountPoint))
			deviceInfo += fmt.Sprintf("└─ [%s%s]\n\n", usedBar, availBar)

			sb.WriteString(deviceInfo)
		}
	}

	return sb.String()
}

// getDiskInfoCmd executes the df command and parses its output
func getDiskInfoCmd() tea.Msg {
	cmd := exec.Command("df", "-k")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	return parseDfOutput(string(output))
}

// parseDfOutput parses the output of the df command
func parseDfOutput(output string) []DiskInfo {
	lines := strings.Split(output, "\n")
	var diskInfos []DiskInfo

	// Skip header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Handle lines that might be wrapped
		if !strings.HasPrefix(line, "/") && !strings.HasPrefix(line, "tmpfs") && !strings.HasPrefix(line, "devtmpfs") {
			continue
		}

		// Split the line into fields
		fields := splitDfLine(line)
		if len(fields) < 6 {
			continue
		}

		// Parse numeric values
		size, _ := strconv.ParseInt(fields[1], 10, 64)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		avail, _ := strconv.ParseInt(fields[3], 10, 64)

		// Parse use percentage
		usePercentStr := strings.TrimSuffix(fields[4], "%")
		usePercent, _ := strconv.Atoi(usePercentStr)

		diskInfos = append(diskInfos, DiskInfo{
			Filesystem: fields[0],
			Size:       size * 1024, // Convert from KB to bytes
			Used:       used * 1024,
			Available:  avail * 1024,
			UsePercent: usePercent,
			MountPoint: fields[5],
		})
	}

	return diskInfos
}

// splitDfLine splits a df output line into fields
func splitDfLine(line string) []string {
	// Use regex to handle variable whitespace
	re := regexp.MustCompile(`\s+`)
	return re.Split(line, -1)
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// Update handles events and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case []DiskInfo:
		m.diskInfos = msg
		m.err = nil
		return m, tea.Quit // Quit after showing the output
	case error:
		m.err = msg
		return m, tea.Quit // Quit if there's an error
	}
	return m, nil
}

func main() {
	p := tea.NewProgram(Model{})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
