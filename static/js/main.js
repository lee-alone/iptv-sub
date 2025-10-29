// 进度条相关逻辑已移除

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

// 启动测试所有频道逻辑改由各页面独立实现

// 启动手动更新
function startManualUpdate() {
    fetch('/update', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            // 更新成功后自动刷新页面，确保频道状态数据同步
            location.reload();
        } else {
            alert('启动更新失败: ' + data.message);
        }
    })
    .catch(error => console.error('启动更新失败:', error));
}

// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', function() {
    // 无全局进度条需要初始化
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
  const apiInput = document.getElementById('apiUrl');
  if (!apiInput) {
    alert('未找到地址输入框');
    return;
  }
  apiInput.select();
  apiInput.setSelectionRange(0, 99999); // 兼容移动端
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(apiInput.value)
        .then(() => {
          alert("地址已复制到剪贴板！");
        })
        .catch(err => {
          document.execCommand('copy');
          alert("地址已复制到剪贴板！");
        });
    } else {
      document.execCommand('copy');
      alert("地址已复制到剪贴板！");
    }
  } catch (err) {
    alert("复制失败，请手动复制地址。");
  }
}

// 统一进度条逻辑已移除
