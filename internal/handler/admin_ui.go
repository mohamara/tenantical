package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AdminUIHandler struct{}

func NewAdminUIHandler() *AdminUIHandler {
	return &AdminUIHandler{}
}

func (h *AdminUIHandler) ServeAdminPanel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	html := `<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>پنل مدیریت Tenantical</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
            direction: rtl;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        
        .header {
            text-align: center;
            color: white;
            margin-bottom: 30px;
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
        }
        
        .header p {
            font-size: 1.1rem;
            opacity: 0.9;
        }
        
        .card {
            background: white;
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 600;
        }
        
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }
        
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .btn {
            padding: 12px 30px;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
        }
        
        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        
        .btn-danger {
            background: #e74c3c;
            color: white;
            padding: 8px 20px;
            font-size: 0.9rem;
        }
        
        .btn-danger:hover {
            background: #c0392b;
        }
        
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        
        .tenants-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        
        .tenants-table th,
        .tenants-table td {
            padding: 15px;
            text-align: right;
            border-bottom: 1px solid #e0e0e0;
        }
        
        .tenants-table th {
            background: #f8f9fa;
            font-weight: 600;
            color: #333;
        }
        
        .tenants-table tr:hover {
            background: #f8f9fa;
        }
        
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #999;
        }
        
        .empty-state svg {
            width: 100px;
            height: 100px;
            margin-bottom: 20px;
            opacity: 0.3;
        }
        
        .alert {
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: none;
        }
        
        .alert-success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        
        .alert-error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        
        .loading {
            display: none;
            text-align: center;
            padding: 20px;
            color: #667eea;
        }
        
        .loading.active {
            display: block;
        }
        
        @media (max-width: 768px) {
            .header h1 {
                font-size: 2rem;
            }
            
            .card {
                padding: 20px;
            }
            
            .tenants-table {
                font-size: 0.9rem;
            }
            
            .tenants-table th,
            .tenants-table td {
                padding: 10px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 پنل مدیریت Tenantical</h1>
            <p>مدیریت Tenants و Domain Routing</p>
        </div>
        
        <div class="card">
            <h2 style="margin-bottom: 20px; color: #333;">افزودن Tenant جدید</h2>
            <div id="alert"></div>
            <form id="tenantForm">
                <div class="form-group">
                    <label for="domain">دامنه (Domain):</label>
                    <input type="text" id="domain" name="domain" placeholder="مثال: tenant1.example.com یا *.example.com" required>
                </div>
                <div class="form-group">
                    <label for="tenant_id">Tenant ID:</label>
                    <input type="text" id="tenant_id" name="tenant_id" placeholder="مثال: tenant-123" required>
                </div>
                <div class="form-group">
                    <label for="project_route">مسیر پروژه (Project Route):</label>
                    <input type="text" id="project_route" name="project_route" placeholder="مثال: /projects/backend (اختیاری)" value="/projects/backend">
                </div>
                <button type="submit" class="btn btn-primary" id="submitBtn">افزودن Tenant</button>
            </form>
        </div>
        
        <div class="card">
            <h2 style="margin-bottom: 20px; color: #333;">لیست Tenants</h2>
            <div class="loading" id="loading">در حال بارگذاری...</div>
            <div id="tenantsContainer"></div>
        </div>
    </div>
    
    <script>
        const API_BASE = '/admin/tenants';
        
        // نمایش پیام
        function showAlert(message, type = 'success') {
            const alertDiv = document.getElementById('alert');
            alertDiv.className = 'alert alert-' + type;
            alertDiv.textContent = message;
            alertDiv.style.display = 'block';
            setTimeout(() => {
                alertDiv.style.display = 'none';
            }, 5000);
        }
        
        // بارگذاری لیست tenants
        async function loadTenants() {
            const container = document.getElementById('tenantsContainer');
            const loading = document.getElementById('loading');
            
            loading.classList.add('active');
            container.innerHTML = '';
            
            try {
                const response = await fetch(API_BASE);
                if (!response.ok) throw new Error('خطا در دریافت اطلاعات');
                
                const data = await response.json();
                const tenants = data.tenants || [];
                
                loading.classList.remove('active');
                
                if (tenants.length === 0) {
                    container.innerHTML = '<div class="empty-state"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" /></svg><p>هیچ tenant ثبت نشده است</p></div>';
                    return;
                }
                
                let tableHTML = '<table class="tenants-table"><thead><tr><th>دامنه</th><th>Tenant ID</th><th>مسیر پروژه</th><th>تاریخ ایجاد</th><th>عملیات</th></tr></thead><tbody>';
                
                tenants.forEach(tenant => {
                    tableHTML += '<tr>' +
                        '<td><strong>' + escapeHtml(tenant.domain) + '</strong></td>' +
                        '<td><code>' + escapeHtml(tenant.tenant_id) + '</code></td>' +
                        '<td><code>' + escapeHtml(tenant.project_route || '/projects/backend') + '</code></td>' +
                        '<td>' + escapeHtml(tenant.created_at || '-') + '</td>' +
                        '<td><button class="btn btn-danger" onclick="deleteTenant(\'' + escapeHtml(tenant.domain) + '\')">حذف</button></td>' +
                        '</tr>';
                });
                
                tableHTML += '</tbody></table>';
                container.innerHTML = tableHTML;
            } catch (error) {
                loading.classList.remove('active');
                container.innerHTML = '<div class="empty-state"><p>خطا در بارگذاری اطلاعات: ' + escapeHtml(error.message) + '</p></div>';
            }
        }
        
        // افزودن tenant جدید
        async function addTenant(e) {
            e.preventDefault();
            
            const submitBtn = document.getElementById('submitBtn');
            const formData = {
                domain: document.getElementById('domain').value.trim(),
                tenant_id: document.getElementById('tenant_id').value.trim(),
                project_route: document.getElementById('project_route').value.trim() || '/projects/backend'
            };
            
            if (!formData.domain || !formData.tenant_id) {
                showAlert('لطفاً فیلدهای اجباری را پر کنید', 'error');
                return;
            }
            
            submitBtn.disabled = true;
            submitBtn.textContent = 'در حال افزودن...';
            
            try {
                const response = await fetch(API_BASE, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(formData)
                });
                
                const data = await response.json();
                
                if (!response.ok) {
                    throw new Error(data.error || data.message || 'خطا در افزودن tenant');
                }
                
                showAlert('Tenant با موفقیت افزوده شد', 'success');
                document.getElementById('tenantForm').reset();
                document.getElementById('project_route').value = '/projects/backend';
                await loadTenants();
            } catch (error) {
                showAlert('خطا: ' + error.message, 'error');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = 'افزودن Tenant';
            }
        }
        
        // حذف tenant
        async function deleteTenant(domain) {
            if (!confirm('آیا از حذف این tenant اطمینان دارید؟\n\n' + domain)) {
                return;
            }
            
            try {
                const encodedDomain = encodeURIComponent(domain);
                const response = await fetch(API_BASE + '/' + encodedDomain, {
                    method: 'DELETE'
                });
                
                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.error || data.message || 'خطا در حذف tenant');
                }
                
                showAlert('Tenant با موفقیت حذف شد', 'success');
                await loadTenants();
            } catch (error) {
                showAlert('خطا: ' + error.message, 'error');
            }
        }
        
        // Escape HTML برای جلوگیری از XSS
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        // رویدادها
        document.getElementById('tenantForm').addEventListener('submit', addTenant);
        
        // بارگذاری اولیه
        loadTenants();
        
        // بارگذاری مجدد هر 30 ثانیه
        setInterval(loadTenants, 30000);
    </script>
</body>
</html>`
	
	w.Write([]byte(html))
}

func (h *AdminUIHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin", h.ServeAdminPanel)
}

