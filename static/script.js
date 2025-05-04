// 全局变量
let currentGroup = 'all';
let channels = [];
const toast = new bootstrap.Toast(document.getElementById('toast'));
const importModal = new bootstrap.Modal(document.getElementById('importProgress'));

// 显示提示消息
function showToast(message, isError = false) {
    const toastElement = document.getElementById('toast');
    toastElement.querySelector('.toast-body').textContent = message;
    toastElement.querySelector('.bi').className = isError ? 'bi bi-exclamation-circle' : 'bi bi-info-circle';
    if (isError) {
        toastElement.classList.add('bg-danger', 'text-white');
    } else {
        toastElement.classList.remove('bg-danger', 'text-white');
    }
    toast.show();
}

// 加载频道列表
async function loadChannels() {
    try {
        const response = await fetch('/api/channels');
        const data = await response.json();
        channels = data.channels;
        updateChannelList();
        updateGroups(data.groups);
        document.getElementById('totalCount').textContent = data.total;
    } catch (error) {
        showToast('加载频道列表失败', true);
    }
}

// 更新频道列表显示
function updateChannelList(searchTerm = '') {
    const channelList = document.getElementById('channelList');
    channelList.innerHTML = '';
    
    const filteredChannels = channels.filter(channel => {
        const matchesGroup = currentGroup === 'all' || channel.group_title === currentGroup;
        const matchesSearch = !searchTerm || 
            channel.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            channel.url.toLowerCase().includes(searchTerm.toLowerCase());
        return matchesGroup && matchesSearch;
    });

    filteredChannels.forEach(channel => {
        const item = document.createElement('div');
        item.className = 'list-group-item';
        item.innerHTML = `
            <div class="d-flex justify-content-between align-items-start">
                <div class="channel-info">
                    <div class="d-flex align-items-center">
                        ${channel.tvg_logo ? `<img src="${channel.tvg_logo}" class="channel-logo me-2" alt="">` : ''}
                        <h6 class="mb-0">${channel.name}</h6>
                    </div>
                    <small class="text-muted d-block">${channel.url}</small>
                    <div class="channel-stats">
                        <span class="badge bg-info me-1">${channel.resolution}</span>
                        <span class="badge bg-secondary me-1">${channel.group_title || '未分类'}</span>
                        <small class="text-muted">最后检查: ${channel.last_check || 'Never'}</small>
                    </div>
                </div>
                <div class="btn-group">
                    <button class="btn btn-sm btn-outline-primary" onclick="copyToClipboard('${channel.url}')">
                        <i class="bi bi-clipboard"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-success" onclick="playChannel('${channel.url}')">
                        <i class="bi bi-play-fill"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteChannel('${channel.url}')">
                        <i class="bi bi-trash"></i>
                    </button>
                </div>
            </div>
        `;
        channelList.appendChild(item);
    });
}

// 更新分组列表
function updateGroups(groups) {
    const groupsList = document.getElementById('channelGroups');
    const exportDropdown = document.querySelector('#exportDropdown + .dropdown-menu');
    
    // 保持"全部频道"选项
    const allChannelsItem = groupsList.firstElementChild;
    groupsList.innerHTML = '';
    groupsList.appendChild(allChannelsItem);
    
    // 添加分组
    groups.forEach(group => {
        const groupItem = document.createElement('a');
        groupItem.href = '#';
        groupItem.className = `list-group-item list-group-item-action ${currentGroup === group ? 'active' : ''}`;
        groupItem.setAttribute('data-group', group);
        groupItem.textContent = group;
        
        // 添加频道计数
        const count = channels.filter(ch => ch.group_title === group).length;
        const badge = document.createElement('span');
        badge.className = 'badge bg-primary float-end';
        badge.textContent = count;
        groupItem.appendChild(badge);
        
        groupsList.appendChild(groupItem);
        
        // 添加到导出下拉菜单
        const exportItem = document.createElement('li');
        exportItem.innerHTML = `<a class="dropdown-item" href="/playlist/group/${encodeURIComponent(group)}" 
                                  download="${group}.m3u8">${group}</a>`;
        exportDropdown.appendChild(exportItem);
    });
    
    // 绑定分组点击事件
    groupsList.querySelectorAll('a').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const group = e.target.getAttribute('data-group');
            currentGroup = group;
            groupsList.querySelectorAll('a').forEach(a => a.classList.remove('active'));
            e.target.classList.add('active');
            updateChannelList();
        });
    });
}

// 复制到剪贴板
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('已复制到剪贴板');
    } catch (err) {
        showToast('复制失败', true);
    }
}

// 播放频道
function playChannel(url) {
    window.open(url, '_blank');
}

// 删除频道
async function deleteChannel(url) {
    if (!confirm('确定要删除这个频道吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`/api/channel/${encodeURIComponent(url)}`, {
            method: 'DELETE'
        });
        const data = await response.json();
        
        if (data.success) {
            showToast('频道已删除');
            loadChannels();
        } else {
            showToast(data.error || '删除失败', true);
        }
    } catch (error) {
        showToast('删除失败', true);
    }
}

// 清理无效频道
async function cleanupChannels() {
    if (!confirm('确定要清理所有无效频道吗？这可能需要一些时间。')) {
        return;
    }
    
    try {
        const response = await fetch('/api/cleanup', {
            method: 'POST'
        });
        const data = await response.json();
        
        if (data.success) {
            showToast('清理完成');
            loadChannels();
        } else {
            showToast('清理失败', true);
        }
    } catch (error) {
        showToast('清理失败', true);
    }
}

// 搜索功能
document.getElementById('searchInput').addEventListener('input', (e) => {
    updateChannelList(e.target.value.trim());
});

// 提交新的M3U8地址
document.getElementById('addUrlForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const urlInput = document.getElementById('m3u8Url');
    const url = urlInput.value.trim();

    // 显示进度弹窗
    const modal = new bootstrap.Modal(document.getElementById('importProgress'));
    const modalBody = document.getElementById('importProgress').querySelector('.modal-body');
    modalBody.innerHTML = `
        <div class="progress mb-3">
            <div class="progress-bar progress-bar-striped progress-bar-animated" 
                 role="progressbar" style="width: 100%"></div>
        </div>
        <div id="importStats">
            <p class="text-center mb-0">正在加载和解析M3U内容，请稍候...</p>
        </div>
    `;
    modal.show();

    try {
        const response = await fetch('/api/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ url }),
        });
        const data = await response.json();
        
        showImportResults(data);
        urlInput.value = '';
    } catch (error) {
        modalBody.innerHTML = `
            <div class="alert alert-danger">
                导入失败：${error.message || '未知错误'}
            </div>
        `;
    }
});

// 显示导入结果
function showImportResults(results) {
    const modalBody = document.querySelector('#importProgress .modal-body');
    
    let content = `
        <div class="import-summary mb-3">
            <h6>导入结果统计</h6>
            <div class="row g-2">
                <div class="col-4">
                    <div class="p-2 border rounded text-center">
                        <div class="h4 mb-0">${results.total}</div>
                        <small class="text-muted">总频道数</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-success bg-opacity-10">
                        <div class="h4 mb-0">${results.added}</div>
                        <small class="text-muted">新增频道</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-warning bg-opacity-10">
                        <div class="h4 mb-0">${results.skipped}</div>
                        <small class="text-muted">重复频道</small>
                    </div>
                </div>
            </div>
        </div>`;

    // 按分组统计
    if (results.details && results.details.length > 0) {
        const groupStats = {};
        results.details.forEach(detail => {
            const group = detail.group || '未分类';
            if (!groupStats[group]) {
                groupStats[group] = { total: 0, added: 0, skipped: 0 };
            }
            groupStats[group].total++;
            if (detail.status === 'added') {
                groupStats[group].added++;
            } else {
                groupStats[group].skipped++;
            }
        });

        content += `
        <div class="group-stats mb-3">
            <h6>分组统计</h6>
            <div class="table-responsive">
                <table class="table table-sm">
                    <thead>
                        <tr>
                            <th>分组</th>
                            <th>总数</th>
                            <th>新增</th>
                            <th>重复</th>
                        </tr>
                    </thead>
                    <tbody>`;
        
        Object.entries(groupStats).forEach(([group, stats]) => {
            content += `
                <tr>
                    <td>${group}</td>
                    <td>${stats.total}</td>
                    <td class="text-success">${stats.added}</td>
                    <td class="text-warning">${stats.skipped}</td>
                </tr>`;
        });
        
        content += `
                    </tbody>
                </table>
            </div>
        </div>`;

        // 显示部分详情
        content += `
        <div class="details-section">
            <h6>新增频道详情 <small class="text-muted">(显示前10个)</small></h6>
            <div class="list-group list-group">`;
        
        const addedChannels = results.details.filter(d => d.status === 'added').slice(0, 10);
        addedChannels.forEach(detail => {
            content += `
                <div class="list-group-item">
                    <div class="d-flex w-100 justify-content-between">
                        <h6 class="mb-1">${detail.name || '未命名频道'}</h6>
                        <small class="text-muted">${detail.group || '未分类'}</small>
                    </div>
                    <small class="text-muted d-block text-truncate">${detail.url}</small>
                </div>`;
        });
        
        content += `</div></div>`;
    }

    modalBody.innerHTML = content;
    
    // 刷新频道列表
    loadChannelList();
}

// 加载频道列表
async function loadChannelList() {
    const response = await fetch('/api/channels');
    const data = await response.json();
    
    // 更新总数
    document.getElementById('totalCount').textContent = data.total;
    
    // 更新分组列表
    const groupsList = document.getElementById('channelGroups');
    const currentGroup = document.querySelector('#channelGroups .active').dataset.group;
    
    // 保持"全部"选项
    groupsList.innerHTML = `
        <a href="#" class="list-group-item list-group-item-action ${currentGroup === 'all' ? 'active' : ''}" 
           data-group="all">
            全部频道 <span class="badge bg-primary float-end">${data.total}</span>
        </a>`;
    
    // 添加分组
    data.groups.sort().forEach(group => {
        const groupChannels = data.channels.filter(ch => ch.group_title === group);
        groupsList.innerHTML += `
            <a href="#" class="list-group-item list-group-item-action ${currentGroup === group ? 'active' : ''}" 
               data-group="${group}">
                ${group} <span class="badge bg-secondary float-end">${groupChannels.length}</span>
            </a>`;
    });
}

// 复制到剪贴板
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('已复制到剪贴板');
    } catch (err) {
        showToast('复制失败', true);
    }
}

// 播放频道
function playChannel(url) {
    window.open(url, '_blank');
}

// 删除频道
async function deleteChannel(url) {
    if (!confirm('确定要删除这个频道吗？')) {
        return;
    }
    
    try {
        const response = await fetch(`/api/channel/${encodeURIComponent(url)}`, {
            method: 'DELETE'
        });
        const data = await response.json();
        
        if (data.success) {
            showToast('频道已删除');
            loadChannels();
        } else {
            showToast(data.error || '删除失败', true);
        }
    } catch (error) {
        showToast('删除失败', true);
    }
}

// 清理无效频道
async function cleanupChannels() {
    if (!confirm('确定要清理所有无效频道吗？这可能需要一些时间。')) {
        return;
    }
    
    try {
        const response = await fetch('/api/cleanup', {
            method: 'POST'
        });
        const data = await response.json();
        
        if (data.success) {
            showToast('清理完成');
            loadChannels();
        } else {
            showToast('清理失败', true);
        }
    } catch (error) {
        showToast('清理失败', true);
    }
}

// 搜索功能
document.getElementById('searchInput').addEventListener('input', (e) => {
    updateChannelList(e.target.value.trim());
});

// 提交新的M3U8地址
document.getElementById('addUrlForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const urlInput = document.getElementById('m3u8Url');
    const url = urlInput.value.trim();

    // 显示进度弹窗
    const modal = new bootstrap.Modal(document.getElementById('importProgress'));
    const modalBody = document.getElementById('importProgress').querySelector('.modal-body');
    modalBody.innerHTML = `
        <div class="progress mb-3">
            <div class="progress-bar progress-bar-striped progress-bar-animated" 
                 role="progressbar" style="width: 100%"></div>
        </div>
        <div id="importStats">
            <p class="text-center mb-0">正在加载和解析M3U内容，请稍候...</p>
        </div>
    `;
    modal.show();

    try {
        const response = await fetch('/api/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ url }),
        });
        const data = await response.json();
        
        showImportResults(data);
        urlInput.value = '';
    } catch (error) {
        modalBody.innerHTML = `
            <div class="alert alert-danger">
                导入失败：${error.message || '未知错误'}
            </div>
        `;
    }
});

// 显示导入结果
function showImportResults(results) {
    const modalBody = document.querySelector('#importProgress .modal-body');
    
    let content = `
        <div class="import-summary mb-3">
            <h6>导入结果统计</h6>
            <div class="row g-2">
                <div class="col-4">
                    <div class="p-2 border rounded text-center">
                        <div class="h4 mb-0">${results.total}</div>
                        <small class="text-muted">总频道数</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-success bg-opacity-10">
                        <div class="h4 mb-0">${results.added}</div>
                        <small class="text-muted">新增频道</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-warning bg-opacity-10">
                        <div class="h4 mb-0">${results.skipped}</div>
                        <small class="text-muted">重复频道</small>
                    </div>
                </div>
            </div>
        </div>`;

    // 按分组统计
    if (results.details && results.details.length > 0) {
        const groupStats = {};
        results.details.forEach(detail => {
            const group = detail.group || '未分类';
            if (!groupStats[group]) {
                groupStats[group] = { total: 0, added: 0, skipped: 0 };
            }
            groupStats[group].total++;
            if (detail.status === 'added') {
                groupStats[group].added++;
            } else {
                groupStats[group].skipped++;
            }
        });

        content += `
        <div class="group-stats mb-3">
            <h6>分组统计</h6>
            <div class="table-responsive">
                <table class="table table-sm">
                    <thead>
                        <tr>
                            <th>分组</th>
                            <th>总数</th>
                            <th>新增</th>
                            <th>重复</th>
                        </tr>
                    </thead>
                    <tbody>`;
        
        Object.entries(groupStats).forEach(([group, stats]) => {
            content += `
                <tr>
                    <td>${group}</td>
                    <td>${stats.total}</td>
                    <td class="text-success">${stats.added}</td>
                    <td class="text-warning">${stats.skipped}</td>
                </tr>`;
        });
        
        content += `
                    </tbody>
                </table>
            </div>
        </div>`;

        // 显示部分详情
        content += `
        <div class="details-section">
            <h6>新增频道详情 <small class="text-muted">(显示前10个)</small></h6>
            <div class="list-group list-group">`;
        
        const addedChannels = results.details.filter(d => d.status === 'added').slice(0, 10);
        addedChannels.forEach(detail => {
            content += `
                <div class="list-group-item">
                    <div class="d-flex w-100 justify-content-between">
                        <h6 class="mb-1">${detail.name || '未命名频道'}</h6>
                        <small class="text-muted">${detail.group || '未分类'}</small>
                    </div>
                    <small class="text-muted d-block text-truncate">${detail.url}</small>
                </div>`;
        });
        
        content += `</div></div>`;
    }

    modalBody.innerHTML = content;
    
    // 刷新频道列表
    loadChannelList();
}

// 加载频道列表
async function loadChannelList() {
    const response = await fetch('/api/channels');
    const data = await response.json();
    
    // 更新总数
    document.getElementById('totalCount').textContent = data.total;
    
    // 更新分组列表
    const groupsList = document.getElementById('channelGroups');
    const currentGroup = document.querySelector('#channelGroups .active').dataset.group;
    
    // 保持"全部"选项
    groupsList.innerHTML = `
        <a href="#" class="list-group-item list-group-item-action ${currentGroup === 'all' ? 'active' : ''}" 
           data-group="all">
            全部频道 <span class="badge bg-primary float-end">${data.total}</span>
        </a>`;
    
    // 添加分组
    data.groups.sort().forEach(group => {
        const groupChannels = data.channels.filter(ch => ch.group_title === group);
        groupsList.innerHTML += `
            <a href="#" class="list-group-item list-group-item-action ${currentGroup === group ? 'active' : ''}" 
               data-group="${group}">
                ${group} <span class="badge bg-secondary float-end">${groupChannels.length}</span>
            </a>`;
    });
}

// 添加通道检查功能
async function checkChannels(autoRepair = false) {
    const modalBody = document.querySelector('#importProgress .modal-body');
    modalBody.innerHTML = `
        <div class="progress mb-3">
            <div class="progress-bar progress-bar-striped progress-bar-animated" 
                 role="progressbar" style="width: 100%"></div>
        </div>
        <div id="checkStats">
            <p class="text-center mb-0">正在检查频道可用性，请稍候...</p>
        </div>
    `;

    const modal = new bootstrap.Modal(document.getElementById('importProgress'));
    modal.show();

    try {
        const response = await fetch('/api/check', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ auto_repair: autoRepair })
        });
        
        const results = await response.json();
        showCheckResults(results);
    } catch (error) {
        modalBody.innerHTML = `
            <div class="alert alert-danger">
                检查失败：${error.message || '未知错误'}
            </div>
        `;
    }
}

// 更新进度显示
function updateProgress(current, total) {
    const progressBar = document.querySelector('#importProgress .progress-bar');
    const percentage = (current / total * 100).toFixed(1);
    progressBar.style.width = `${percentage}%`;
    progressBar.setAttribute('aria-valuenow', percentage);
    progressBar.textContent = `${percentage}%`;
}

// 并发检查频道
async function checkChannelsConcurrent() {
    const modalBody = document.querySelector('#importProgress .modal-body');
    modalBody.innerHTML = `
        <div class="check-progress mb-3">
            <div class="d-flex justify-content-between mb-2">
                <span>检查进度</span>
                <span id="progressText">准备中...</span>
            </div>
            <div class="progress">
                <div class="progress-bar progress-bar-striped progress-bar-animated" 
                     role="progressbar" style="width: 0%" aria-valuenow="0">
                </div>
            </div>
        </div>
        <div class="current-status mt-3">
            <div class="card">
                <div class="card-body p-2">
                    <div class="d-flex justify-content-between">
                        <div>已完成: <span id="checkedCount">0</span></div>
                        <div>正常: <span id="availableCount" class="text-success">0</span></div>
                        <div>异常: <span id="unavailableCount" class="text-danger">0</span></div>
                    </div>
                </div>
            </div>
        </div>
        <div id="checkStats" class="mt-3">
            <small class="text-muted">正在启动并发检查...</small>
        </div>
    `;

    const modal = new bootstrap.Modal(document.getElementById('importProgress'));
    modal.show();

    // 启动检查
    try {
        const response = await fetch('/api/check/concurrent', {
            method: 'POST',
        });
        const results = await response.json();
        
        // 更新状态卡片
        document.getElementById('checkedCount').textContent = results.total;
        document.getElementById('availableCount').textContent = results.available;
        document.getElementById('unavailableCount').textContent = 
            (results.unavailable || results.failed || 0);
        
        // 显示最终结果
        showCheckResults(results, true);
    } catch (error) {
        modalBody.innerHTML = `
            <div class="alert alert-danger">
                检查失败：${error.message || '未知错误'}
            </div>
        `;
    }
}

// 添加工具函数
function formatTime(seconds) {
    if (seconds < 60) return `${seconds.toFixed(1)}秒`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = (seconds % 60).toFixed(1);
    return `${minutes}分${remainingSeconds}秒`;
}

// 更新检查结果显示
function showCheckResults(results, isConcurrent = false) {
    const modalBody = document.querySelector('#importProgress .modal-body');
    
    let content = `
        <div class="check-summary mb-3">
            <h6>检查结果统计 ${isConcurrent ? '(并发检查)' : ''}</h6>
            <div class="row g-2">
                <div class="col-4">
                    <div class="p-2 border rounded text-center">
                        <div class="h5 mb-0">${results.total}</div>
                        <small class="text-muted">总数</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-success bg-opacity-10">
                        <div class="h5 mb-0">${results.available}</div>
                        <small class="text-muted">正常</small>
                    </div>
                </div>
                <div class="col-4">
                    <div class="p-2 border rounded text-center bg-danger bg-opacity-10">
                        <div class="h5 mb-0">${results.unavailable || results.failed || 0}</div>
                        <small class="text-muted">异常</small>
                    </div>
                </div>
            </div>
        </div>`;

    // 添加详细信息表格
    if (results.details && results.details.length > 0) {
        content += `
        <div class="details-section">
            <h6>检查详情</h6>
            <div class="table-responsive">
                <table class="table table-sm">
                    <thead>
                        <tr>
                            <th>频道</th>
                            <th>状态</th>
                            <th>响应时间</th>
                        </tr>
                    </thead>
                    <tbody>`;
        
        results.details.forEach(detail => {
            const statusClass = detail.status === '正常' ? 'success' : 'danger';
            content += `
                <tr>
                    <td>${detail.name || '未命名'}</td>
                    <td><span class="badge bg-${statusClass}">${detail.status}</span></td>
                    <td><small>${detail.message || '-'}</small></td>
                </tr>`;
        });
        
        content += `
                    </tbody>
                </table>
            </div>
        </div>`;
    }

    // 添加性能统计
    if (results.performance) {
        content += `
        <div class="performance-stats mt-3">
            <h6>性能统计</h6>
            <div class="row g-2">
                <div class="col-6 col-md-4">
                    <div class="p-2 border rounded">
                        <small class="d-block text-muted">平均响应时间</small>
                        <strong>${results.performance.avgResponseTime.toFixed(2)}秒</strong>
                    </div>
                </div>
                <div class="col-6 col-md-4">
                    <div class="p-2 border rounded">
                        <small class="d-block text-muted">总耗时</small>
                        <strong>${formatTime(results.performance.totalTime)}</strong>
                    </div>
                </div>
                <div class="col-6 col-md-4">
                    <div class="p-2 border rounded">
                        <small class="d-block text-muted">并发数</small>
                        <strong>${results.performance.concurrency}</strong>
                    </div>
                </div>
            </div>
        </div>`;
    }

    modalBody.innerHTML = content;
}

// 导出检查报告
function exportCheckReport(results) {
    const reportContent = `IPTV频道检查报告
时间：${new Date().toLocaleString()}

检查结果统计：
总数：${results.total}
正常：${results.available}
异常：${results.unavailable || 0}
已修复：${results.repaired || 0}
失败：${results.failed || 0}

详细信息：
${results.details.map(detail => `
频道：${detail.name || '未命名频道'}
状态：${detail.status}
${detail.new_url ? `新地址：${detail.new_url}` : ''}
${detail.error ? `错误信息：${detail.error}` : ''}`).join('\n')}`;

    const blob = new Blob([reportContent], { type: 'text/plain;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `iptv_check_report_${new Date().toISOString().split('T')[0]}.txt`;
    a.click();
    URL.revokeObjectURL(a.href);
}

// 页面加载时初始化
document.addEventListener('DOMContentLoaded', () => {
    loadChannels();
});
