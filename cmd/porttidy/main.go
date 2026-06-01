package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iiwish/porttidy/pkg/config"
	"github.com/iiwish/porttidy/pkg/killer"
	"github.com/iiwish/porttidy/pkg/model"
	"github.com/iiwish/porttidy/pkg/output"
	"github.com/iiwish/porttidy/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	cfgPath  string
	jsonMode bool
	verbose  bool
)

const exitCandidatesFound = 2

var rootCmd = &cobra.Command{
	Use:   "porttidy",
	Short: "安全清理 AI 开发会话遗留的本地 dev server",
	Long: `Porttidy 扫描配置目录下的本地开发服务，
保守识别可安全清理的孤儿进程，避免误杀编辑器、终端、浏览器和 agent runtime。`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "扫描开发进程",
	RunE:  runScan,
}

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "专家模式：关闭匹配的开发进程",
	RunE:  runKill,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "安全清理可确认的孤儿开发进程",
	RunE:  runCleanup,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("porttidy v0.1.0")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&jsonMode, "json", "j", false, "JSON 输出")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")

	// scan flags
	scanCmd.Flags().Bool("orphan", false, "只显示孤儿进程")
	scanCmd.Flags().Int("port", 0, "过滤指定端口")
	scanCmd.Flags().Int("pid", 0, "过滤指定 PID")
	scanCmd.Flags().Duration("since", 0, "只显示运行超过多久的进程 (e.g. 1h, 30m)")

	// kill flags
	killCmd.Flags().Bool("all", false, "关闭所有扫描到的开发进程")
	killCmd.Flags().Bool("orphan", false, "只关闭孤儿进程")
	killCmd.Flags().String("dir", "", "关闭指定目录下的进程")
	killCmd.Flags().Int("port", 0, "关闭占用指定端口的进程")
	killCmd.Flags().Int("pid", 0, "关闭指定 PID")
	killCmd.Flags().Duration("since", 0, "关闭运行超过指定时间的进程")
	killCmd.Flags().Bool("force", false, "跳过确认（用于自动化/脚本）")
	killCmd.Flags().Bool("dry-run", false, "只列出，不执行")

	// cleanup flags
	cleanupCmd.Flags().String("dir", "", "只清理指定目录下的安全候选")
	cleanupCmd.Flags().Int("port", 0, "只清理占用指定端口的安全候选")
	cleanupCmd.Flags().Int("pid", 0, "只清理指定 PID 的安全候选")
	cleanupCmd.Flags().Duration("since", 0, "只清理运行超过指定时间的安全候选")
	cleanupCmd.Flags().Bool("force", false, "跳过确认（用于自动化/脚本）")
	cleanupCmd.Flags().Bool("dry-run", false, "只列出，不执行")

	rootCmd.AddCommand(scanCmd, cleanupCmd, killCmd, versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *exitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func exitWithCode(code int) error {
	return &exitCodeError{code: code}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置: %w", err)
	}
	return cfg, nil
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "扫描目录: %v\n", cfg.TargetDirs)
	}

	opts := scanner.ScanOptions{
		OrphanOnly: cmd.Flags().Changed("orphan") && mustBool(cmd.Flags().GetBool("orphan")),
		Port:       mustInt(cmd.Flags().GetInt("port")),
		PID:        mustInt(cmd.Flags().GetInt("pid")),
		Since:      mustDuration(cmd.Flags().GetDuration("since")),
	}

	s := scanner.New(cfg)
	result, err := s.Scan(opts)
	if err != nil {
		return fmt.Errorf("扫描失败: %w", err)
	}

	if jsonMode {
		if err := output.PrintJSON(result); err != nil {
			return err
		}
	} else {
		output.PrintTable(result)
	}

	if opts.OrphanOnly && result.Summary.Total > 0 {
		return exitWithCode(exitCandidatesFound)
	}

	return nil
}

func runKill(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dryRun := mustBool(cmd.Flags().GetBool("dry-run"))
	force := mustBool(cmd.Flags().GetBool("force"))

	all := mustBool(cmd.Flags().GetBool("all"))
	if all && force {
		return fmt.Errorf("--all 不能和 --force 同用；自动化请使用 cleanup --force")
	}

	scanOpts := scanner.ScanOptions{
		OrphanOnly: cmd.Flags().Changed("orphan") && mustBool(cmd.Flags().GetBool("orphan")),
		Port:       mustInt(cmd.Flags().GetInt("port")),
		PID:        mustInt(cmd.Flags().GetInt("pid")),
		Since:      mustDuration(cmd.Flags().GetDuration("since")),
	}

	if all {
		scanOpts.OrphanOnly = false
	}

	s := scanner.New(cfg)
	result, err := s.Scan(scanOpts)
	if err != nil {
		return fmt.Errorf("扫描失败: %w", err)
	}

	targets := selectTargets(result, targetSelection{
		DirFilter: mustString(cmd.Flags().GetString("dir")),
		SafeOnly:  force,
	})

	if all && !dryRun {
		if !confirmExpertKillAll(targets) {
			return nil
		}
		force = true
	}

	return executeTargets(targets, dryRun, force, all, cfg.TargetDirs)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dryRun := mustBool(cmd.Flags().GetBool("dry-run"))
	force := mustBool(cmd.Flags().GetBool("force"))

	scanOpts := scanner.ScanOptions{
		OrphanOnly: true,
		Port:       mustInt(cmd.Flags().GetInt("port")),
		PID:        mustInt(cmd.Flags().GetInt("pid")),
		Since:      mustDuration(cmd.Flags().GetDuration("since")),
	}

	s := scanner.New(cfg)
	result, err := s.Scan(scanOpts)
	if err != nil {
		return fmt.Errorf("扫描失败: %w", err)
	}

	targets := selectTargets(result, targetSelection{
		DirFilter: mustString(cmd.Flags().GetString("dir")),
		SafeOnly:  true,
	})

	return executeTargets(targets, dryRun, force, false, cfg.TargetDirs)
}

type targetSelection struct {
	DirFilter string
	SafeOnly  bool
}

func selectTargets(result *model.ScanResult, opts targetSelection) []model.Process {
	var targets []model.Process
	for _, proj := range result.Projects {
		for _, p := range proj.Processes {
			if opts.DirFilter != "" && !strings.Contains(p.CWD, opts.DirFilter) {
				continue
			}
			if opts.SafeOnly && !p.CanForceCleanup {
				continue
			}
			targets = append(targets, p)
		}
	}
	return targets
}

func executeTargets(targets []model.Process, dryRun, force, allowUnsafe bool, targetDirs []string) error {
	if len(targets) == 0 {
		if jsonMode {
			return output.PrintKillResult([]model.KillResult{})
		}
		fmt.Println("没有找到可清理的进程。")
		return nil
	}

	if !jsonMode {
		fmt.Printf("发现 %d 个进程：\n", len(targets))
		for i, p := range targets {
			ports := ""
			if len(p.Ports) > 0 {
				ports = fmt.Sprintf(" (端口 %v)", p.Ports)
			}
			fmt.Printf("  [%d] PID %d  %s%s  %s  [%s] %s\n",
				i+1,
				p.PID,
				p.Name,
				ports,
				output.ShortenPath(p.CWD, 30),
				output.SafetyLabel(p),
				output.ShortenText(output.ProcessReason(p), 60))
		}
		fmt.Println()
	}

	if dryRun {
		if jsonMode {
			if err := output.PrintKillResult(killer.New(targetDirs...).Kill(targets, true)); err != nil {
				return err
			}
			return exitWithCode(exitCandidatesFound)
		}
		fmt.Println("🛑 --dry-run 模式，不执行关闭。")
		return exitWithCode(exitCandidatesFound)
	}

	if !force {
		fmt.Print("确认关闭以上进程？(y/N) ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("已取消。")
			return nil
		}
	}

	k := killer.New(targetDirs...)
	if allowUnsafe {
		k = killer.NewUnsafe(targetDirs...)
	}
	results := k.Kill(targets, false)

	if jsonMode {
		return output.PrintKillResult(results)
	}

	fmt.Println("\n关闭结果：")
	for _, r := range results {
		switch r.Status {
		case "killed":
			fmt.Printf("  ✓ PID %d 已关闭\n", r.PID)
		case "already_dead":
			fmt.Printf("  ⚠ PID %d 已经不存在\n", r.PID)
		case "failed":
			fmt.Printf("  ✗ PID %d 失败: %s\n", r.PID, r.Error)
		}
	}

	return nil
}

func confirmExpertKillAll(targets []model.Process) bool {
	if len(targets) == 0 || jsonMode {
		return true
	}

	fmt.Println("警告：kill --all 是专家模式，可能包含仍在使用的开发进程。")
	fmt.Print("如确认继续，请输入 kill all: ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if confirm != "kill all" {
		fmt.Println("已取消。")
		return false
	}
	return true
}

// must helpers for flag parsing
func mustBool(v bool, err error) bool {
	if err != nil {
		panic(err)
	}
	return v
}

func mustInt(v int, err error) int {
	if err != nil {
		panic(err)
	}
	return v
}

func mustDuration(v time.Duration, err error) time.Duration {
	if err != nil {
		panic(err)
	}
	return v
}

func mustString(v string, err error) string {
	if err != nil {
		panic(err)
	}
	return v
}
