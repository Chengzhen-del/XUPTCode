// js/config.js - 全局配置文件
// 统一管理所有HTML共用的配置项
window.AppConfig = {
    // API基础地址（核心：所有HTML都用这个变量）
    API_BASE_URL: 'http://192.168.137.1:8080',
    // 其他可共用的配置（可选，按需添加）
    USER_INFO_API: '/staff/get-info',
    RESOURCE_LIST_API: '/resource/list',
    STORAGE_KEY: 'xuptcode_user_info',
    LOGIN_PAGE_URL: '/page/login',
    // 分页默认值
    DEFAULT_PAGE_SIZE: 10,
    // 提示框默认时长
    TOAST_DURATION: 3000
};
// config.js - AI页面核心配置
const AI_CONFIG = {
    // DeepSeek API基础配置
    API_BASE_URL: "https://api.deepseek.com/v1/chat/completions",
    API_MODEL: "deepseek-chat", // 对话模型
    API_MAX_TOKENS: 2048,       // 最大回复长度
    API_TEMPERATURE: 0.7,       // 回复随机性（0-1）
    API_STREAM: false,          // 是否流式返回

    // 页面默认配置
    DEFAULT_TITLE: "代码库AI助手",
    EMPTY_TIPS: "在这里提问代码相关问题，AI会为你解答～",
    // 可在这里配置默认API Key（建议仅测试用，生产环境从后端获取）
    DEFAULT_API_KEY: "",

    // 样式配置（贴合代码库风格）
    THEME: {
        primary: "#2563eb",      // 主色调（代码库常用深蓝）
        secondary: "#1e293b",    // 辅助色（深灰）
        codeBg: "#1f2937",       // 代码背景色
        textPrimary: "#f8fafc",  // 浅色文本
        textSecondary: "#94a3b8" // 次要文本
    }
};