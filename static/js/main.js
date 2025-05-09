// 定期更新测试进度
function updateTestProgress() {    fetch('/channels/test-progress')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const progress = data.progress;
                const percent = (progress.completed / progress.total * 100).toFixed(1);
                
                // 更新进度条
                const progressBar = document.getElementById('test-progress-bar');
                if (progressBar) {
                    progressBar.style.width = percent + '%';
                    progressBar.textContent = `${percent}% (${progress.completed}/${progress.total})`;
                }
                
                // 更新状态统计
                const statsDiv = document.getElementById('test-stats');
                if (statsDiv) {
                    if (!data.progress.is_testing) {
                        // 测试已完成，显示最终结果
                        statsDiv.innerHTML = `
                            测试完成！总计: ${progress.total} | 
                            在线: ${progress.online} | 
                            离线: ${progress.offline} | 
                            开始时间: ${progress.start_time} |
                            最后测试时间: ${formatDateTime(new Date())}
                        `;
                    } else {
                        // 测试进行中，显示实时进度
                        statsDiv.innerHTML = `
                            在线: ${progress.online} | 
                            离线: ${progress.offline} | 
                            开始时间: ${progress.start_time}
                        `;
                        // 仅在测试进行中继续更新
                    }
                }
            }
        })
        .catch(error => console.error('更新测试进度失败:', error));
}

// 删除订阅源确认
function confirmDeleteSubscription(url) {
    if (confirm('确定要删除这个订阅源吗？')) {
        fetch('/subscriptions/delete/' + encodeURIComponent(url), {
            method: 'POST'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                location.reload();
            } else {
                alert('删除失败: ' + data.message);
            }
        })
        .catch(error => console.error('删除订阅源失败:', error));
    }
}

// 删除导出文件确认
function confirmDeleteExport(filename) {
    if (confirm('确定要删除这个导出文件吗？')) {
        fetch('/export/delete/' + encodeURIComponent(filename), {
            method: 'POST'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                location.reload();
            } else {
                alert('删除失败');
            }
        })
        .catch(error => console.error('删除导出文件失败:', error));
    }
}

// 测试单个频道
function testChannel(channelId) {
    fetch('/channels/test/' + channelId, {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // 更新状态标签
            const statusBadge = document.getElementById('channel-status-' + channelId);
            if (statusBadge) {
                statusBadge.className = 'status-badge status-' + data.status;
                statusBadge.textContent = data.status.toUpperCase();
            }
        } else {
            alert('测试失败: ' + data.message);
        }
    })
    .catch(error => console.error('测试频道失败:', error));
}

// 启动测试所有频道
function startTestAll() {
    fetch('/channels/test-all', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // 显示进度条
            const progressSection = document.getElementById('test-progress-section');
            if (progressSection) {
                progressSection.style.display = 'block';
            }
            // 开始定期更新进度
            updateTestProgress();
        } else {
            alert('启动测试失败: ' + data.message);
        }
    })
    .catch(error => console.error('启动测试失败:', error));
}

// 启动手动更新
function startManualUpdate() {
    fetch('/update', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('更新任务已启动');
        } else {
            alert('启动更新失败: ' + data.message);
        }
    })
    .catch(error => console.error('启动更新失败:', error));
}

// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', function() {
    // 如果存在进度条，开始定期更新进度
    const progressSection = document.getElementById('test-progress-section');
    if (progressSection && progressSection.style.display !== 'none') {
        updateTestProgress();
    }
});

// 格式化日期时间
function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN');
}

// 表单验证
function validateForm(formId) {
    const form = document.getElementById(formId);
    if (form) {
        const requiredFields = form.querySelectorAll('[required]');
        let isValid = true;
        
        requiredFields.forEach(field => {
            if (!field.value.trim()) {
                isValid = false;
                field.classList.add('is-invalid');
            } else {
                field.classList.remove('is-invalid');
            }
        });
        
        return isValid;
    }
    return true;
}

function copyApiUrl() {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(window.location.href)
      .then(() => {
        alert("地址已复制到剪贴板！");
      })
      .catch(err => {
        console.error('复制失败: ', err);
        alert("复制失败，请手动复制地址。");
      });
  } else {
    const tempInput = document.createElement("input");
    tempInput.value = window.location.href;
    document.body.appendChild(tempInput);
    tempInput.select();
    document.execCommand("copy");
    document.body.removeChild(tempInput);
    alert("地址已复制到剪贴板！");
  }
}
