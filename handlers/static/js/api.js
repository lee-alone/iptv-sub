/**
 * API 封装模块
 * 提供与后端 API 交互的方法
 */

const api = {
    baseURL: '/api',

    /**
     * 发送 HTTP 请求
     */
    async request(method, endpoint, data = null) {
        const url = `${this.baseURL}${endpoint}`;
        const options = {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(url, options);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`API Error [${method} ${endpoint}]:`, error);
            throw error;
        }
    },

    /**
     * 获取所有订阅源
     */
    async getSubscriptions() {
        const result = await this.request('GET', '/subscriptions');
        return result.data || [];
    },

    /**
     * 添加订阅源
     */
    async addSubscription(url, name, enabled = true) {
        return await this.request('POST', '/subscriptions', {
            url,
            name,
            enabled,
        });
    },

    /**
     * 删除订阅源
     */
    async deleteSubscription(url) {
        return await this.request('DELETE', `/subscriptions?url=${encodeURIComponent(url)}`);
    },

    /**
     * 更新单个订阅源
     */
    async updateSubscription(url, name, enabled = true) {
        return await this.request('PUT', `/subscriptions?url=${encodeURIComponent(url)}`, {
            name,
            enabled,
        });
    },

    /**
     * 更新所有订阅源
     */
    async updateSubscriptions() {
        return await this.request('POST', '/subscriptions/update');
    },

    /**
     * 获取所有频道
     */
    async getChannels(onlyOnline = false) {
        const endpoint = onlyOnline ? '/channels?online=true' : '/channels';
        const result = await this.request('GET', endpoint);
        return result.data || [];
    },

    /**
     * 获取单个频道
     */
    async getChannel(id) {
        const result = await this.request('GET', `/channels/${id}`);
        return result.data;
    },

    /**
     * 聚合频道
     */
    async aggregateChannels(subscriptionUrl, matchBy = 'name', threshold = 0.85) {
        return await this.request('POST', '/aggregate', {
            subscription_url: subscriptionUrl,
            match_by: matchBy,
            threshold: threshold,
        });
    },

    /**
     * 测试频道
     */
    async testChannels(testAllSources = false) {
        return await this.request('POST', '/test', {
            test_all_sources: testAllSources,
        });
    },

    /**
     * 测试单个频道
     */
    async testChannel(id) {
        return await this.request('POST', `/channels/${id}/test`);
    },

    /**
     * 导出频道
     */
    async exportChannels(format = 'm3u', onlyWorking = true) {
        return await this.request('POST', '/export', {
            format: format,
            only_working: onlyWorking,
        });
    },

    /**
     * 获取统计信息
     */
    async getStats() {
        const result = await this.request('GET', '/stats');
        return result;
    },

    /**
     * 健康检查
     */
    async healthCheck() {
        try {
            const response = await fetch('/health');
            return response.ok;
        } catch {
            return false;
        }
    },

    /**
     * 获取配置
     */
    async getConfig() {
        const result = await this.request('GET', '/config');
        return result.data || {};
    },

    /**
     * 更新配置
     */
    async updateConfig(settings) {
        return await this.request('PUT', '/config', settings);
    },

    /**
     * 重启服务器
     */
    async restart() {
        return await this.request('POST', '/restart');
    },

    /**
     * 导出订阅源为JSON
     */
    async exportSubscriptions() {
        return await this.request('GET', '/subscriptions/export');
    },

    /**
     * 导入订阅源
     */
    async importSubscriptions(subscriptions) {
        return await this.request('POST', '/subscriptions/import', {
            subscriptions: subscriptions,
        });
    },
};
