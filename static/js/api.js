// 后端API基础地址（前后端分离时替换为后端域名）
const BASE_URL = AppConfig.API_BASE_URL;

/**
 * 处理空值字段（适配MySQL DATE类型）
 * @param {Object} data 原始请求数据
 * @returns {Object} 处理后的数据（空字符串转null）
 */
function handleEmptyFields(data) {
    // 深拷贝数据，避免修改原对象
    const processedData = { ...data };

    // 核心适配：birth_date为空字符串/undefined时转为null
    if (processedData.birth_date === '' || processedData.birth_date === undefined) {
        processedData.birth_date = null;
    }

    // 可选：过滤掉所有值为null/undefined的字段（也可保留null，二选一）
    // Object.keys(processedData).forEach(key => {
    //   if (processedData[key] === null || processedData[key] === undefined) {
    //     delete processedData[key];
    //   }
    // });

    return processedData;
}


    async function handleRegister() {
    const username = document.getElementById("username").value.trim();
    const password = document.getElementById("password").value.trim();
    const email = document.getElementById("email").value.trim();
    const phone = document.getElementById("phone").value.trim();
    const role = document.getElementById("role").value;
    const errorTip = document.getElementById("errorTip");

    // 1️⃣ 前端校验
    if (!username) {
    errorTip.innerText = "请输入用户名";
    return;
}
    if (username.length < 3 || username.length > 50) {
    errorTip.innerText = "用户名长度需在 3-50 位之间";
    return;
}
    if (!password) {
    errorTip.innerText = "请输入密码";
    return;
}
    if (password.length < 6 || password.length > 20) {
    errorTip.innerText = "密码长度需在 6-20 位之间";
    return;
}

    errorTip.innerText = "";

    // 2️⃣ 构造请求体（严格匹配后端 DTO）
    const registerData = {
    username,
    password,
    role
};

    if (email) registerData.email = email;
    if (phone) registerData.phone = phone;

    try {
    const res = await register(registerData);

    // 3️⃣ 成功判断（对齐后端）
    if (res.code === 200) {
    alert("注册成功，即将跳转到登录页面");
    window.location.href = "/page/login";
} else {
    // 4️⃣ 错误信息兜底
    errorTip.innerText = res.message || "注册失败";
}
} catch (err) {
    console.error("注册异常：", err);
    errorTip.innerText = "网络异常，请稍后重试";
}
}


/**
 * 登录接口请求
 * @param {Object} data 登录参数（username/phone/email + password）
 * @returns {Promise} 响应结果
 */
/**
 * 退出登录接口请求
 * @param {string} token JWT令牌
 * @returns {Promise} 响应结果
 */
async function logout(token) {
    const response = await fetch(`${BASE_URL}/staff/logout`, {
        method: "GET", // 若后端要求POST，可改为POST
        headers: {
            "Authorization": `Bearer ${token}`,
        },
    });
    return await response.json();
}
// 通用请求函数（携带 Token）
async function requestWithToken(url, options = {}) {
    const token = localStorage.getItem("token");
    if (!token) {
        // Token 不存在，跳转登录页
        window.location.href = "/page/login";
        throw new Error("Token 已过期，请重新登录");
    }

    // 合并 headers，添加 Token
    options.headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`, // 匹配后端 JWT 中间件的 Token 格式
        ...options.headers,
    };

    return fetch(url, options);
}

// 示例：调用更新用户接口
async function updateUser(userData) {
    return requestWithToken('/api/user', {
        method: 'PUT',
        body: JSON.stringify(userData),
    });
}