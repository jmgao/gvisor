package(licenses = ["notice"])

load("//tools/go_stateify:defs.bzl", "go_library", "go_test")

go_library(
    name = "proc",
    srcs = [
        "cgroup.go",
        "cpuinfo.go",
        "exec_args.go",
        "fds.go",
        "filesystems.go",
        "fs.go",
        "inode.go",
        "loadavg.go",
        "meminfo.go",
        "mounts.go",
        "net.go",
        "proc.go",
        "rpcinet_proc.go",
        "stat.go",
        "sys.go",
        "sys_net.go",
        "sys_net_state.go",
        "task.go",
        "uid_gid_map.go",
        "uptime.go",
        "version.go",
    ],
    importpath = "gvisor.googlesource.com/gvisor/pkg/sentry/fs/proc",
    visibility = ["//pkg/sentry:internal"],
    deps = [
        "//pkg/abi/linux",
        "//pkg/log",
        "//pkg/sentry/context",
        "//pkg/sentry/fs",
        "//pkg/sentry/fs/fsutil",
        "//pkg/sentry/fs/proc/device",
        "//pkg/sentry/fs/proc/seqfile",
        "//pkg/sentry/fs/ramfs",
        "//pkg/sentry/inet",
        "//pkg/sentry/kernel",
        "//pkg/sentry/kernel/auth",
        "//pkg/sentry/kernel/kdefs",
        "//pkg/sentry/kernel/time",
        "//pkg/sentry/limits",
        "//pkg/sentry/mm",
        "//pkg/sentry/socket",
        "//pkg/sentry/socket/rpcinet",
        "//pkg/sentry/socket/unix",
        "//pkg/sentry/socket/unix/transport",
        "//pkg/sentry/usage",
        "//pkg/sentry/usermem",
        "//pkg/syserror",
        "//pkg/waiter",
    ],
)

go_test(
    name = "proc_test",
    size = "small",
    srcs = [
        "net_test.go",
        "sys_net_test.go",
    ],
    embed = [":proc"],
    deps = [
        "//pkg/abi/linux",
        "//pkg/sentry/context",
        "//pkg/sentry/inet",
        "//pkg/sentry/usermem",
    ],
)
