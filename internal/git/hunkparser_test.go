package git

import (
	"sort"
	"testing"
)

// ============================================================================
// TC-T1.2-01: Go 文件 diff —— 提取函数上下文
// ============================================================================
func TestExtractHunkContexts_GoFile_ExtractsFuncName(t *testing.T) {
	// Arrange: 模拟 Go 文件的 diff 输出，hunk header 末尾带函数名。
	diff := `diff --git a/server.go b/server.go
index abc..def 100644
--- a/server.go
+++ b/server.go
@@ -10,7 +10,8 @@ func doSomething(ctx context.Context) error {
 	// some code
+	new line
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	contexts := result["server.go"]
	if len(contexts) != 1 {
		t.Fatalf("期望 1 条上下文，实际: %d", len(contexts))
	}
	expected := "func doSomething(ctx context.Context) error {"
	if contexts[0] != expected {
		t.Errorf("上下文 = %q, want %q", contexts[0], expected)
	}
}

// ============================================================================
// TC-T1.2-02: Python 文件 diff —— 提取函数上下文
// ============================================================================
func TestExtractHunkContexts_PythonFile_ExtractsFuncName(t *testing.T) {
	// Arrange
	diff := `diff --git a/utils.py b/utils.py
--- a/utils.py
+++ b/utils.py
@@ -5,3 +5,4 @@ def calculate_total(items):
     total = sum(items)
+    return total
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	contexts := result["utils.py"]
	if len(contexts) != 1 {
		t.Fatalf("期望 1 条上下文，实际: %d", len(contexts))
	}
	expected := "def calculate_total(items):"
	if contexts[0] != expected {
		t.Errorf("上下文 = %q, want %q", contexts[0], expected)
	}
}

// ============================================================================
// TC-T1.2-03: JS 文件 diff —— 提取函数上下文
// ============================================================================
func TestExtractHunkContexts_JSFile_ExtractsFuncName(t *testing.T) {
	// Arrange
	diff := `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -20,5 +20,6 @@ export function handleClick() {
     console.log("clicked");
+    updateUI();
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	contexts := result["app.js"]
	if len(contexts) != 1 {
		t.Fatalf("期望 1 条上下文，实际: %d", len(contexts))
	}
	expected := "export function handleClick() {"
	if contexts[0] != expected {
		t.Errorf("上下文 = %q, want %q", contexts[0], expected)
	}
}

// ============================================================================
// TC-T1.2-04: 冷门语言（Git 无内置 funcname 规则）—— 空上下文，不报错
// ============================================================================
func TestExtractHunkContexts_NoFuncName_EmptyContext(t *testing.T) {
	// Arrange: .proto 文件 Git 通常无 funcname 规则，hunk header 末尾无文本。
	diff := `diff --git a/service.proto b/service.proto
--- a/service.proto
+++ b/service.proto
@@ -1,2 +1,3 @@
 syntax = "proto3";
+option go_package = "./pb";
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	contexts := result["service.proto"]
	if len(contexts) != 1 {
		t.Fatalf("期望 1 条上下文（空字符串），实际: %d", len(contexts))
	}
	// Git 对 .proto 无内置规则，上下文应为空字符串。
	if contexts[0] != "" {
		t.Errorf("冷门语言上下文应为空字符串，实际: %q", contexts[0])
	}
}

// ============================================================================
// TC-T1.2-05: 同一文件多个 hunk 落在同一函数内 → 去重
// ============================================================================
func TestExtractHunkContexts_SameFunction_Deduplicates(t *testing.T) {
	// Arrange: 同一文件内 3 个 hunk，都落在 handleRequest 函数中。
	diff := `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -10,5 +10,6 @@ func handleRequest(w http.ResponseWriter, r *http.Request) {
 	// parse input
+	validateInput(r)
@@ -30,5 +31,6 @@ func handleRequest(w http.ResponseWriter, r *http.Request) {
 	// process
+	logRequest(r)
@@ -50,3 +51,4 @@ func handleRequest(w http.ResponseWriter, r *http.Request) {
 	// respond
+	setHeaders(w)
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	contexts := result["handler.go"]
	// 3 个 hunk，但上下文文本相同，去重后仅剩 1 条。
	if len(contexts) != 1 {
		t.Fatalf("期望去重后仅 1 条上下文，实际: %d (values: %v)", len(contexts), contexts)
	}
}

// ============================================================================
// TC-T1.2-06: 输入不含 @@ 标记 → 返回空 map，不报错
// ============================================================================
func TestExtractHunkContexts_NoHunkHeader_ReturnsEmpty(t *testing.T) {
	// Arrange: 输入是纯文本，不含任何 diff 标记。
	diff := "some random text\nwithout any diff markers\n"

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	if len(result) != 0 {
		t.Errorf("无 @@ 标记时应返回空 map，实际长度: %d", len(result))
	}
}

// ============================================================================
// 补充: 空输入
// ============================================================================
func TestExtractHunkContexts_EmptyInput_ReturnsNil(t *testing.T) {
	// Arrange: 空字符串输入。

	// Act
	result := ExtractHunkContexts("")

	// Assert
	if result != nil {
		t.Errorf("空输入应返回 nil map，实际: %v", result)
	}
}

// ============================================================================
// 补充: 多文件混合场景
// ============================================================================
func TestExtractHunkContexts_MultipleFiles(t *testing.T) {
	// Arrange: 一次 diff 中同时包含 Go 和 Python 文件。
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -5,3 +5,4 @@ func main() {
 	// init
+	setup()
diff --git a/helper.py b/helper.py
--- a/helper.py
+++ b/helper.py
@@ -1,2 +1,3 @@ def helper():
     pass
+    return True
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert
	if len(result) != 2 {
		t.Fatalf("期望 2 个文件，实际: %d", len(result))
	}

	// 验证各文件的上下文正确提取。
	goCtx, ok := result["main.go"]
	if !ok {
		t.Error("结果应包含 main.go")
	}
	if len(goCtx) != 1 || goCtx[0] != "func main() {" {
		t.Errorf("main.go 上下文不正确: %v", goCtx)
	}

	pyCtx, ok := result["helper.py"]
	if !ok {
		t.Error("结果应包含 helper.py")
	}
	if len(pyCtx) != 1 || pyCtx[0] != "def helper():" {
		t.Errorf("helper.py 上下文不正确: %v", pyCtx)
	}
}

// ============================================================================
// 补充: diff 中不同文件有相同函数名时不应跨文件去重
// ============================================================================
func TestExtractHunkContexts_SameFuncDifferentFiles_KeepsBoth(t *testing.T) {
	// Arrange: 两个不同文件中都有 main 函数。
	diff := `diff --git a/cmd/a/main.go b/cmd/a/main.go
--- a/cmd/a/main.go
+++ b/cmd/a/main.go
@@ -3,3 +3,4 @@ func main() {
 	// a
+	a()
diff --git a/cmd/b/main.go b/cmd/b/main.go
--- a/cmd/b/main.go
+++ b/cmd/b/main.go
@@ -3,3 +3,4 @@ func main() {
 	// b
+	b()
`

	// Act
	result := ExtractHunkContexts(diff)

	// Assert: 两个文件各有一条，不应跨文件去重。
	if len(result) != 2 {
		t.Fatalf("期望 2 个文件，实际: %d", len(result))
	}

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if keys[0] != "cmd/a/main.go" || keys[1] != "cmd/b/main.go" {
		t.Errorf("文件路径不正确: %v", keys)
	}
}
