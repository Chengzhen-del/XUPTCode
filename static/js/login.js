async function login(loginData) {
    const baseUrl = window.location.origin; // 自动获取当前域名，避免硬编码
    try {
        const response = await fetch(`${baseUrl}/api/login`, { // 替换为实际的登录接口路径
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // 若后端需要跨域凭证，添加以下行
                // 'Credentials': 'include'
            },
            body: JSON.stringify(loginData),
        });

        // 处理 HTTP 状态码非 200 的情况（比如 401/500）
        if (!response.ok) {
            throw new Error(`请求失败：${response.status} ${response.statusText}`);
        }

        return await response.json();
    } catch (err) {
        console.error("登录请求失败：", err);
        throw err; // 抛出错误到外层处理
    }
}
// 登录按钮点击事件
async function handleLogin() {
    const account = document.getElementById("account").value.trim();
    const password = document.getElementById("password").value.trim();
    const errorTip = document.getElementById("errorTip");

    // 1️⃣ 前端参数校验
    if (!account) {
        errorTip.innerText = "请输入用户名 / 手机号 / 邮箱";
        return;
    }
    if (!password) {
        errorTip.innerText = "请输入密码";
        return;
    }

    errorTip.innerText = "";

    // 2️⃣ 构造请求参数（严格匹配后端 DTO）
    const loginData = { password };
    if (/^1[3-9]\d{9}$/.test(account)) {
        loginData.phone = account;
    } else if (/^[\w.-]+@[\w.-]+\.\w+$/.test(account)) {
        loginData.email = account;
    } else {
        loginData.username = account;
    }

    try {
        // 3️⃣ 调用登录接口
        const res = await login(loginData);

        // 安全校验：确保 res 是合法对象
        if (typeof res !== 'object' || res === null) {
            errorTip.innerText = "登录失败：响应格式异常";
            return;
        }

        // 4️⃣ 正确判断后端响应（code=200 为成功）
        if (res.code === 200) {
            // 从 res.message 中解构数据（核心修正：替换 res.data → res.message）
            const { token, user_id, username, role } = res.message || {};

            // 校验 Token 必传
            if (!token) {
                errorTip.innerText = "登录失败：Token 缺失";
                return;
            }

            // 5️⃣ 存储完整登录态（补充 user_id/role，后续接口可能用到）
            localStorage.setItem("token", token);
            localStorage.setItem("username", username || "");
            localStorage.setItem("user_id", user_id || "");
            localStorage.setItem("role", role || "");

            // 6️⃣ 登录成功跳转
            window.location.href = "/page/index";
        } else {
            // 处理后端返回的错误信息（兼容 message 是字符串/对象的情况）
            const errorMsg = typeof res.message === 'string'
                ? res.message
                : (res.message?.error || "登录失败：账号或密码错误");
            errorTip.innerText = errorMsg;
        }
    } catch (err) {
        console.error("登录异常：", err);
        // 区分网络错误和接口错误
        errorTip.innerText = err.message.includes('Failed to fetch')
            ? "网络异常，请检查连接后重试"
            : "登录异常，请稍后重试";
    }
}
// 完整的登录请求封装（独立函数）
