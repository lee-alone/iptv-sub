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
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    },
};

// 页面加载完成后的初始化
document.addEventListener('DOMContentLoaded', function() {
    // 检查 API 连接
    api.healthCheck().then(ok => {
        if (!ok) {
            utils.showError('无法连接到服务器');
        }
    });
});

// 导出工具函数
window.utils = utils;
