load("//tools:build_rules/shims.bzl", "go_library", "go_test")

package(default_visibility = ["//kythe:default_visibility"])

go_library(
    name = "proxy",
    srcs = ["proxy.go"],
    deps = [
        "//kythe/go/services/graphstore",
        "//kythe/go/storage/gsutil",
        "//kythe/go/util/compare",
        "//kythe/proto:storage_go_proto",
    ],
)

go_test(
    name = "proxy_test",
    size = "small",
    srcs = ["proxy_test.go"],
    library = "proxy",
    visibility = ["//visibility:private"],
    deps = [
        "//kythe/go/services/graphstore",
        "//kythe/proto:storage_go_proto",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
