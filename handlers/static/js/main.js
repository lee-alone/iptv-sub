/**
 * 主要 JavaScript 文件
 * 包含通用工具函数和页面交互逻辑
 */

// 工具函数
const utils = {
    /**
     * 格式化日期时间
     */
    formatDateTime(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleString('zh-CN');
    },

    /**
     * 格式化文件大小
     */
    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    },

    /**
     * 显示成功提示
     */
    showSuccess(message) {
        this.showAlert(message, 'success');
    },

    /**
     * 显示错误提示
     */
    showError(message) {
        this.showAlert(message, 'danger');
    },

    /**
     * 显示警告提示
     */
    showWarning(message) {
        this.showAlert(message, 'warning');
    },

    /**
     * 显示普通提示
     */
    showInfo(message) {
        this.showAlert(message, 'info');
    },

    /**
     * 显示提示信息
     */
    showAlert(message, type = 'info') {
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
        alertDiv.role = 'alert';
        alertDiv.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;

        const container = document.querySelector('.container-fluid') || document.body;
        container.insertBefore(alertDiv, container.firstChild);

        // 3 秒后自动关闭
        setTimeout(() => {
            alertDiv.remove();
        }, 3000);
    },

    /**
     * 复制到剪贴板
     */
    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showSuccess('已复制到剪贴板');
        } catch (err) {
            console.error('复制失败:', err);
            this.showError('复制失败');
        }
    },

    /**
     * 确认对话框
     */
    confirm(message) {
        return window.confirm(message);
    },

    /**
     * 获取 URL 参数
     */
    getQueryParam(name) {
        const params = new URLSearchParams(window.location.search);
        return params.get(name);
    },

    /**
     * 设置 URL 参数
     */
    setQueryParam(name, value) {
        const params = new URLSearchParams(window.location.search);
        params.set(name, value);
        window.history.replaceState({}, '', `${window.location.pathname}?${params}`);
    },

    /**
     * 验证 URL
     */
    isValidUrl(string) {
        try {
            new URL(string);
            return true;
        } catch (_) {
            return false;
        }
    },

    /**
     * 防抖函数
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    /**
     * 节流函数
     */
    throttle(func, limit) {
        let inThrottle;
        return function (...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    },
};

// 页面加载完成后的初始化
document.addEventListener('DOMContentLoaded', function () {
    // 检查 API 连接
    api.healthCheck().then(ok => {
        if (!ok) {
            utils.showError('无法连接到服务器');
        }
    });

    // 初始化配置管理
    configManager.init();
});

// 配置管理模块
const configManager = {
    config: null,
    initPromise: null,

    /**
     * 初始化配置
     */
    async init() {
        if (this.initPromise) {
            return this.initPromise;
        }

        this.initPromise = (async () => {
            try {
                this.config = await api.getConfig();
                this.updateUI();
                console.log('configManager initialized:', this.config);
            } catch (error) {
                console.error('Failed to load config:', error);
            }
        })();

        return this.initPromise;
    },

    /**
     * 更新 UI 中的服务器地址信息
     */
    updateUI() {
        if (!this.config) return;

        // 更新所有显示服务器地址的元素
        const serverAddressElements = document.querySelectorAll('[data-server-address]');
        serverAddressElements.forEach(el => {
            el.textContent = this.config.server_address || 'N/A';
        });

        // 更新所有显示播放列表 URL 的元素
        const playlistUrlElements = document.querySelectorAll('[data-playlist-url]');
        playlistUrlElements.forEach(el => {
            el.textContent = this.config.playlist_url || 'N/A';
            el.href = this.config.playlist_url || '#';
        });

        // 更新所有显示本机 IP 的元素
        const localIPElements = document.querySelectorAll('[data-local-ip]');
        localIPElements.forEach(el => {
            el.textContent = this.config.local_ip || 'N/A';
        });

        // 更新所有显示端口的元素
        const portElements = document.querySelectorAll('[data-port]');
        portElements.forEach(el => {
            el.textContent = this.config.port || 'N/A';
        });
    },

    /**
     * 获取服务器地址
     */
    getServerAddress() {
        return this.config?.server_address || 'http://localhost:8080';
    },

    /**
     * 获取播放列表 URL
     */
    getPlaylistURL() {
        return this.config?.playlist_url || 'http://localhost:8080/playlist.m3u';
    },

    /**
     * 获取本机 IP
     */
    getLocalIP() {
        return this.config?.local_ip || 'localhost';
    },

    /**
     * 获取端口
     */
    getPort() {
        return this.config?.port || 8080;
    },
};

// 导出配置管理器
window.configManager = configManager;
