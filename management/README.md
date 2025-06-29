# 🔐 Hướng dẫn chi tiết Blockchain Permissions System

> **Hệ thống phân quyền cho Blockchain private với Geth 1.13.15**
> 
> Tài liệu này hướng dẫn chi tiết cách sử dụng hệ thống phân quyền đã được setup.

---

## 📋 Mục lục

1. [Tổng quan hệ thống](#tổng-quan-hệ-thống)
2. [Cấu trúc files và quyền](#cấu-trúc-files-và-quyền)
3. [Các scripts chính](#các-scripts-chính)
4. [Workflows thường dùng](#workflows-thường-dùng)
5. [Quản lý users](#quản-lý-users)
6. [Roles và permissions](#roles-và-permissions)
7. [Troubleshooting](#troubleshooting)
8. [Best practices](#best-practices)

---

## 🎯 Tổng quan hệ thống

### Những gì đã được setup:

- ✅ **Nhóm blockchain-admins** - Quản lý quyền Linux level
- ✅ **IPC Socket permissions** - Bảo mật kết nối local
- ✅ **Role-based access control** - Phân quyền theo vai trò
- ✅ **User management system** - Quản lý users và permissions
- ✅ **Security APIs** - Chỉ accessible qua IPC

### Kiến trúc bảo mật:

```
┌─────────────────────────────────────────┐
│              Users & Roles              │
├─────────────────────────────────────────┤
│ 👑 Super Admin    - Toàn quyền  |       │
│ ⛏️  Mining Admin   - Điều khiển đào     │
│ 🛡️  Security Admin - Quản lý bảo mật    │
│ 🌐 Network Admin  - Quản lý mạng        │
│ 👀 Read Only      - Chỉ đọc dữ liệu     │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           Permission Layer              │
├─────────────────────────────────────────┤
│ • File permissions (775)                │
│ • Group permissions (blockchain-admins) │
│ • IPC socket (660)                      │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│             Geth Node                   │
├─────────────────────────────────────────┤
│ HTTP APIs:  eth, net, web3 (Public)     │
│ IPC APIs:   All APIs (Admin only)       │
│ Security:   IPC only                    │
└─────────────────────────────────────────┘
```

---

## 📁 Cấu trúc files và quyền

### Files chính:

```bash
📁 management/
├── kiem_tra_quyen.sh      # 🔍 Script kiểm tra quyền
├── roles_blockchain.sh    # 🎭 Console theo role
├── quan_ly_users.sh       # 👥 Quản lý users
└── 📁 ~/ethereum/
    ├── permissions.conf   # 📋 File cấu hình quyền users
    └── 📁 node1/
        └── geth.ipc      # 🔌 Socket IPC
```

### Permissions hiện tại:

```bash
# Kiểm tra quyền thư mục
ls -la ~/ethereum/
# drwxrwxr-x ... liuhinphat blockchain-admins

# Kiểm tra quyền IPC socket  
ls -la ~/ethereum/node1/geth.ipc
# srw-rw---- ... liuhinphat blockchain-admins

# Kiểm tra nhóm user
groups
# blockchain-admins adm dialout cdrom floppy sudo ...
```

---

## 🔧 Các scripts chính

### 1. Script kiểm tra quyền

**File:** `kiem_tra_quyen.sh`

**Chức năng:** Kiểm tra tình trạng quyền hiện tại

```bash
# Cách chạy
./management/kiem_tra_quyen.sh

# Kết quả hiển thị:
# - Thông tin user và nhóm
# - Quyền thư mục blockchain  
# - Trạng thái IPC socket
# - Test kết nối IPC
# - Các APIs có sẵn
# - Tổng kết quyền (Super Admin/Limited)
```

**Khi nào dùng:**
- ✅ Kiểm tra setup ban đầu
- ✅ Troubleshooting quyền
- ✅ Verify sau khi thay đổi cấu hình

---

### 2. Script console theo role

**File:** `roles_blockchain.sh`

**Chức năng:** Mở Geth console với context theo role

```bash
# Xem menu các role
./management/roles_blockchain.sh

# Mở console theo role cụ thể
./management/roles_blockchain.sh super-admin      # 👑 Toàn quyền
./management/roles_blockchain.sh mining-admin     # ⛏️  Chỉ đào
./management/roles_blockchain.sh security-admin   # 🛡️  Chỉ bảo mật
./management/roles_blockchain.sh network-admin    # 🌐 Chỉ mạng
./management/roles_blockchain.sh read-only        # 👀 Chỉ đọc

# Thoát console: Ctrl+D
```

**Trong mỗi console sẽ có:**
- Hướng dẫn lệnh được phép
- Cảnh báo lệnh không được phép
- Kết nối trực tiếp IPC

---

### 3. Script quản lý users

**File:** `quan_ly_users.sh`

**Chức năng:** Quản lý users và phân quyền

```bash
# Mở menu quản lý
./management/quan_ly_users.sh

# Menu options:
# 1. Xem tất cả users và quyền
# 2. Thêm user mới
# 3. Xóa user  
# 5. Kiểm tra quyền của user
# 6. Tạo user Linux + phân quyền
# 7. Xem file cấu hình
# 8. Backup file cấu hình
# 0. Thoát
```

**Interface features:**
- 🎨 Màu sắc phân biệt roles
- ✅ Validation input
- 🛡️ Confirmation cho thao tác nguy hiểm
- 📝 Backup tự động

---

## 🔄 Workflows thường dùng

### Workflow 1: Kiểm tra tình trạng hàng ngày

```bash
# Bước 1: Kiểm tra quyền system
./management/kiem_tra_quyen.sh

# Bước 2: Xem users đã phân quyền
./management/quan_ly_users.sh
# → Chọn 1 (Xem tất cả users)

# Bước 3: Kiểm tra blockchain status
./management/roles_blockchain.sh super-admin
# Trong console:
eth.blockNumber          # Block hiện tại
eth.mining              # Trạng thái mining
txpool.status           # Transaction pool
admin.peers             # Network peers

### Workflow 2: Thêm admin mới

```bash
# Bước 1: Mở quản lý users
./management/quan_ly_users.sh

# Bước 2: Thêm user
# → Chọn 2 (Thêm user mới)
# → Nhập username: "admin_nguyen"
# → Mô tả: "Quản lý đào của team"

# Bước 3: Verify
# → Chọn 1 (Xem tất cả users)
# → Kiểm tra user đã xuất hiện

# Bước 4: Test quyền
# → Chọn 5 (Kiểm tra quyền)
# → Nhập: "admin_nguyen"
```

### Workflow 3: Làm việc theo role

```bash
# Mining tasks
./management/roles_blockchain.sh mining-admin
# Lệnh trong console:
miner.start(1)                    # Bắt đầu đào
miner.stop()                      # Dừng đào
miner.setEtherbase("0x...")       # Đổi địa chỉ thưởng
eth.mining                        # Check status

# Security tasks  
./management/roles_blockchain.sh security-admin
# Lệnh trong console:
security.getWhitelist()           # Xem whitelist
security.addToWhitelist("0x...")  # Thêm address
security.getBlacklist()           # Xem blacklist
security.checkAddress("0x...")    # Kiểm tra address

# Network tasks
./management/roles_blockchain.sh network-admin  
# Lệnh trong console:
admin.peers                       # Xem peers
admin.addPeer("enode://...")      # Thêm peer
admin.nodeInfo                    # Info node
net.peerCount                     # Số peers

---

## 👥 Quản lý users

### File cấu hình: `~/ethereum/permissions.conf`

**Format:** `username:role:description`

```bash
# Ví dụ cấu hình:
liuhinphat:super-admin:Người tạo blockchain
admin_duc:mining-admin:Quản lý đào team 1
admin_mai:security-admin:Chuyên viên bảo mật
guest_analyst:read-only:Phân tích viên dữ liệu
```

### Thêm user manual:

```bash
# Mở file cấu hình
nano ~/ethereum/permissions.conf

# Thêm dòng mới (bỏ dấu # nếu có)
new_user:role:description

# Ví dụ:
admin_long:network-admin:Kỹ sư mạng senior
```

### Xem users nhanh:

```bash
# Hiển thị users đang active
grep -v "^#" ~/ethereum/permissions.conf | grep -v "^$"

# Format đẹp
echo "👥 USERS HIỆN TẠI:"
grep -v "^#" ~/ethereum/permissions.conf | grep -v "^$" | while IFS=: read -r user role desc; do
    echo "👤 $user -> $role ($desc)"
done
```

---

## 🎭 Roles và permissions

### 1. 👑 Super Admin

**Quyền:**
- ✅ Quản lý tài khoản (`personal.*`)
- ✅ Điều khiển đào (`miner.*`) 
- ✅ Quản trị node (`admin.*`)
- ✅ Quản lý bảo mật (`security.*`)
- ✅ Đọc blockchain (`eth.*`)

**Lệnh quan trọng:**
```javascript
// Account management
personal.unlockAccount("0x...", "password", 300)
personal.newAccount("password")

// Mining control  
miner.start(1)
miner.stop()
miner.setEtherbase("0x...")

// Security management
security.addToWhitelist("0x...")
security.addToBlacklist("0x...")

// Network admin
admin.addPeer("enode://...")
admin.peers
```

---

### 2. ⛏️ Mining Admin

**Quyền:**
- ✅ Điều khiển đào (`miner.*`)
- ✅ Đọc blockchain (`eth.*`)
- ❌ Quản lý tài khoản
- ❌ Quản trị node  
- ❌ Quản lý bảo mật

**Lệnh được phép:**
```javascript
// Mining operations
miner.start(1)                    // Bắt đầu đào
miner.stop()                      // Dừng đào
miner.setEtherbase("0x...")       // Đổi địa chỉ nhận thưởng

// Monitoring
eth.mining                        // Check mining status
eth.hashrate                      // Hash rate
eth.blockNumber                   // Block hiện tại
eth.coinbase                      // Địa chỉ miner
```

---

### 3. 🛡️ Security Admin  

**Quyền:**
- ✅ Quản lý bảo mật (`security.*`)
- ✅ Đọc blockchain (`eth.*`)
- ❌ Quản lý tài khoản
- ❌ Điều khiển đào
- ❌ Quản trị node

**Lệnh được phép:**
```javascript
// Whitelist management
security.getWhitelist()           // Xem danh sách trắng
security.addToWhitelist("0x...")  // Thêm vào whitelist
security.removeFromWhitelist("0x...") // Xóa khỏi whitelist

// Blacklist management  
security.getBlacklist()           // Xem danh sách đen
security.addToBlacklist("0x...")  // Thêm vào blacklist
security.removeFromBlacklist("0x...") // Xóa khỏi blacklist

// Address checking
security.checkAddress("0x...")    // Kiểm tra status address
security.isAllowed("0x...")       // Kiểm tra cho phép giao dịch
```

---

### 4. 🌐 Network Admin

**Quyền:**
- ✅ Quản lý mạng (`admin.peers`, `admin.addPeer`, etc.)
- ✅ Đọc blockchain (`eth.*`)
- ❌ Quản lý tài khoản
- ❌ Điều khiển đào
- ❌ Quản lý bảo mật

**Lệnh được phép:**
```javascript
// Peer management
admin.peers                       // Xem tất cả peers
admin.addPeer("enode://...")      // Thêm peer mới
admin.removePeer("enode://...")   // Xóa peer

// Node information
admin.nodeInfo                    // Thông tin node
admin.nodeInfo.enode              // Enode string

// Network status
net.peerCount                     // Số lượng peers
net.listening                     // Trạng thái listening
net.version                       // Network ID
```

---

### 5. 👀 Read Only

**Quyền:**
- ✅ Đọc blockchain (`eth.*`)
- ❌ Tất cả thao tác thay đổi

**Lệnh được phép:**
```javascript
// Blockchain reading
eth.blockNumber                   // Số block hiện tại
eth.getBalance("0x...")           // Xem số dư address
eth.getBlock(number)              // Thông tin block
eth.getTransaction("0x...")       // Thông tin transaction
eth.accounts                      // Danh sách accounts
eth.gasPrice                      // Gas price hiện tại

// Network info
net.version                       // Network ID
net.peerCount                     // Số peers (readonly)
```

---

## 🔧 Troubleshooting

### Vấn đề 1: Không kết nối được IPC

**Triệu chứng:**
```
❌ Không thể kết nối IPC
```

**Giải pháp:**
```bash
# Check Geth có đang chạy không
ps aux | grep geth

# Check IPC socket
ls -la ~/ethereum/node1/geth.ipc

# Check permissions
./kiem_tra_quyen.sh

# Restart Geth nếu cần
# [Geth startup command]
```

---

### Vấn đề 2: Permission denied

**Triệu chứng:**
```
Permission denied when accessing geth.ipc
```

**Giải pháp:**
```bash
# Check nhóm user
groups | grep blockchain-admins

# Nếu không có nhóm:
newgrp blockchain-admins

# Fix permissions
chmod 660 ~/ethereum/node1/geth.ipc
sudo chown $USER:blockchain-admins ~/ethereum/node1/geth.ipc
```

---

### Vấn đề 3: Scripts không chạy

**Triệu chứng:**
```
bash: ./script.sh: Permission denied
```

**Giải pháp:**
```bash
# Cấp quyền thực thi
chmod +x kiem_tra_quyen.sh
chmod +x roles_blockchain.sh  
chmod +x quan_ly_users.sh

# Hoặc chạy bằng bash
bash kiem_tra_quyen.sh
```

---

### Vấn đề 4: User không tìm thấy trong permissions.conf

**Triệu chứng:**
```
❌ User username không có quyền nào được cấu hình
```

**Giải pháp:**
```bash
# Check file có tồn tại không
ls -la ~/ethereum/permissions.conf

# Xem nội dung file
cat ~/ethereum/permissions.conf

# Thêm user nếu chưa có
echo "username:role:description" >> ~/ethereum/permissions.conf
```

---

## 💡 Best practices

### 1. Security

- ✅ **Luôn dùng roles tối thiểu** - Chỉ cấp quyền cần thiết
- ✅ **Regular backup** - Backup permissions.conf thường xuyên  
- ✅ **Monitor access** - Kiểm tra logs truy cập IPC
- ✅ **Strong passwords** - Dùng passwords mạnh cho accounts
- ⚠️ **Cẩn thận với Super Admin** - Chỉ cấp khi thực sự cần

### 2. Operations

- ✅ **Daily checks** - Chạy `kiem_tra_quyen.sh` mỗi ngày
- ✅ **Document changes** - Ghi lại mọi thay đổi permissions
- ✅ **Test trước production** - Test permissions trên staging
- ✅ **Regular cleanup** - Xóa users không còn cần thiết

### 3. User Management

- ✅ **Descriptive usernames** - Dùng tên user có ý nghĩa
- ✅ **Clear descriptions** - Mô tả rõ vai trò user
- ✅ **Regular reviews** - Review danh sách users định kỳ
- ✅ **Offboarding process** - Quy trình xóa user khi nghỉ việc

---

## 📞 Support & Contact

### Tự troubleshoot:

1. **Chạy diagnostics:**
   ```bash
   ./management/kiem_tra_quyen.sh
   ```

2. **Kiểm tra logs:**
   ```bash
   journalctl -u geth -f
   ```

3. **Backup trước khi sửa:**
   ```bash
   ./management/quan_ly_users.sh
   # Chọn 8 (Backup)
   ```

### Emergency commands:

```bash
# Reset permissions nhanh
sudo chown -R $USER:blockchain-admins ~/ethereum/
chmod -R 775 ~/ethereum/
chmod 660 ~/ethereum/node1/geth.ipc

# Backup cấu hình  
cp ~/ethereum/permissions.conf ~/ethereum/permissions.backup.$(date +%Y%m%d_%H%M%S)

# Reset về Super Admin
echo "$(whoami):super-admin:Emergency admin" >> ~/ethereum/permissions.conf
```

---

## 📚 Appendix

### Useful commands:

```bash
# Quick status check
./kiem_tra_quyen.sh | tail -10

# List active users
grep -v "^#" ~/ethereum/permissions.conf | cut -d: -f1

# Count users by role  
grep -v "^#" ~/ethereum/permissions.conf | cut -d: -f2 | sort | uniq -c

# Find user role
grep "^username:" ~/ethereum/permissions.conf | cut -d: -f2
```

### File locations:

- **Scripts:** Current directory
- **Permissions config:** `~/ethereum/permissions.conf`
- **IPC socket:** `~/ethereum/node1/geth.ipc`
- **Geth data:** `~/ethereum/node1/`
- **Logs:** System journal hoặc geth logs

---

> **📝 Tài liệu này được cập nhật:** $(date)
> 
> **📧 Liên hệ support:** [Your contact info]
> 
> **🔄 Version:** 1.0.0