#!/bin/bash

PERMISSION_FILE="$HOME/ethereum/permissions.conf"

# Màu sắc
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Hàm hiển thị menu
show_menu() {
    echo -e "${BLUE}👥 QUẢN LÝ USERS BLOCKCHAIN${NC}"
    echo "=========================="
    echo ""
    echo "1. Xem tất cả users và quyền"
    echo "2. Thêm user mới"
    echo "3. Xóa user"
    echo "4. Thay đổi quyền user"
    echo "5. Kiểm tra quyền của user"
    echo "6. Tạo user Linux + phân quyền"
    echo "7. Xem file cấu hình"
    echo "8. Backup file cấu hình"
    echo "0. Thoát"
    echo ""
}

# Hàm xem tất cả users
list_users() {
    echo -e "${GREEN}📋 DANH SÁCH USERS VÀ QUYỀN${NC}"
    echo "================================="
    echo ""
    
    if [ ! -f "$PERMISSION_FILE" ]; then
        echo -e "${RED}❌ File cấu hình không tồn tại!${NC}"
        return 1
    fi
    
    echo -e "${YELLOW}Format: Username | Role | Mô tả${NC}"
    echo "----------------------------------------"
    
    # Đọc file và hiển thị users (bỏ qua comment và dòng trống)
    grep -v "^#" "$PERMISSION_FILE" | grep -v "^$" | while IFS=: read -r username role description; do
        if [ ! -z "$username" ]; then
            # Tô màu theo role
            case "$role" in
                "super-admin")
                    echo -e "${RED}👑 $username${NC} | $role | $description"
                    ;;
                "mining-admin")
                    echo -e "${YELLOW}⛏️  $username${NC} | $role | $description"
                    ;;
                "security-admin")
                    echo -e "${GREEN}🛡️  $username${NC} | $role | $description"
                    ;;
                "network-admin")
                    echo -e "${BLUE}🌐 $username${NC} | $role | $description"
                    ;;
                "read-only")
                    echo -e "👀 $username | $role | $description"
                    ;;
                *)
                    echo -e "❓ $username | $role | $description"
                    ;;
            esac
        fi
    done
    
    echo ""
    echo -e "${YELLOW}Tổng số users đang hoạt động:${NC} $(grep -v "^#" "$PERMISSION_FILE" | grep -v "^$" | wc -l)"
}

# Hàm thêm user mới
add_user() {
    echo -e "${GREEN}➕ THÊM USER MỚI${NC}"
    echo "=================="
    echo ""
    
    read -p "Nhập username: " username
    if [ -z "$username" ]; then
        echo -e "${RED}❌ Username không được để trống!${NC}"
        return 1
    fi
    
    # Kiểm tra user đã tồn tại chưa
    if grep -q "^$username:" "$PERMISSION_FILE"; then
        echo -e "${RED}❌ User $username đã tồn tại!${NC}"
        return 1
    fi
    
    echo ""
    echo "Chọn role:"
    echo "1. super-admin     👑 (Toàn quyền)"
    echo "2. mining-admin    ⛏️  (Điều khiển đào)"
    echo "3. security-admin  🛡️  (Quản lý bảo mật)"
    echo "4. network-admin   🌐 (Quản lý mạng)"
    echo "5. read-only       👀 (Chỉ đọc)"
    echo ""
    read -p "Chọn role (1-5): " role_choice
    
    case "$role_choice" in
        1) role="super-admin" ;;
        2) role="mining-admin" ;;
        3) role="security-admin" ;;
        4) role="network-admin" ;;
        5) role="read-only" ;;
        *) echo -e "${RED}❌ Lựa chọn không hợp lệ!${NC}"; return 1 ;;
    esac
    
    read -p "Nhập mô tả cho user: " description
    if [ -z "$description" ]; then
        description="User được thêm vào $(date)"
    fi
    
    # Thêm user vào file
    echo "$username:$role:$description" >> "$PERMISSION_FILE"
    echo -e "${GREEN}✅ Đã thêm user $username với role $role${NC}"
}

# Hàm xóa user
remove_user() {
    echo -e "${RED}🗑️  XÓA USER${NC}"
    echo "============="
    echo ""
    
    list_users
    echo ""
    read -p "Nhập username cần xóa: " username
    
    if [ -z "$username" ]; then
        echo -e "${RED}❌ Username không được để trống!${NC}"
        return 1
    fi
    
    # Kiểm tra user có tồn tại không
    if ! grep -q "^$username:" "$PERMISSION_FILE"; then
        echo -e "${RED}❌ User $username không tồn tại!${NC}"
        return 1
    fi
    
    # Xác nhận xóa
    read -p "Bạn có chắc muốn xóa user $username? (y/N): " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        # Xóa user khỏi file
        sed -i "/^$username:/d" "$PERMISSION_FILE"
        echo -e "${GREEN}✅ Đã xóa user $username${NC}"
    else
        echo "Hủy bỏ xóa user"
    fi
}

# Hàm kiểm tra quyền user
check_user_permission() {
    echo -e "${BLUE}🔍 KIỂM TRA QUYỀN USER${NC}"
    echo "======================="
    echo ""
    
    read -p "Nhập username cần kiểm tra: " username
    
    if [ -z "$username" ]; then
        username=$(whoami)
        echo "Sử dụng user hiện tại: $username"
    fi
    
    # Tìm user trong file
    user_info=$(grep "^$username:" "$PERMISSION_FILE")
    
    if [ -z "$user_info" ]; then
        echo -e "${RED}❌ User $username không có quyền nào được cấu hình${NC}"
        return 1
    fi
    
    # Parse thông tin user
    role=$(echo "$user_info" | cut -d: -f2)
    description=$(echo "$user_info" | cut -d: -f3)
    
    echo -e "${GREEN}📊 Thông tin user:${NC}"
    echo "Username: $username"
    echo "Role: $role"
    echo "Mô tả: $description"
    echo ""
    
    # Hiển thị quyền cụ thể
    case "$role" in
        "super-admin")
            echo -e "${RED}👑 SUPER ADMIN - Toàn quyền${NC}"
            echo "✅ Quản lý tài khoản (personal.*)"
            echo "✅ Điều khiển đào (miner.*)"
            echo "✅ Quản trị node (admin.*)"
            echo "✅ Quản lý bảo mật (security.*)"
            echo "✅ Đọc blockchain (eth.*)"
            ;;
        "mining-admin")
            echo -e "${YELLOW}⛏️  MINING ADMIN${NC}"
            echo "✅ Điều khiển đào (miner.*)"
            echo "✅ Đọc blockchain (eth.*)"
            echo "❌ Quản lý tài khoản"
            echo "❌ Quản trị node"
            echo "❌ Quản lý bảo mật"
            ;;
        "security-admin")
            echo -e "${GREEN}🛡️  SECURITY ADMIN${NC}"
            echo "✅ Quản lý bảo mật (security.*)"
            echo "✅ Đọc blockchain (eth.*)"
            echo "❌ Quản lý tài khoản"
            echo "❌ Điều khiển đào"
            echo "❌ Quản trị node"
            ;;
        "network-admin")
            echo -e "${BLUE}🌐 NETWORK ADMIN${NC}"
            echo "✅ Quản lý mạng (admin.peers, admin.addPeer...)"
            echo "✅ Đọc blockchain (eth.*)"
            echo "❌ Quản lý tài khoản"
            echo "❌ Điều khiển đào"
            echo "❌ Quản lý bảo mật"
            ;;
        "read-only")
            echo "👀 READ ONLY"
            echo "✅ Đọc blockchain (eth.*)"
            echo "❌ Tất cả thao tác thay đổi"
            ;;
        *)
            echo -e "${RED}❓ Role không xác định: $role${NC}"
            ;;
    esac
}

# Main menu
while true; do
    show_menu
    read -p "Chọn tùy chọn (0-8): " choice
    echo ""
    
    case $choice in
        1) list_users ;;
        2) add_user ;;
        3) remove_user ;;
        4) echo "Chức năng thay đổi quyền - coming soon!" ;;
        5) check_user_permission ;;
        6) echo "Chức năng tạo user Linux - coming soon!" ;;
        7) echo "File cấu hình: $PERMISSION_FILE"; cat "$PERMISSION_FILE" ;;
        8) cp "$PERMISSION_FILE" "$PERMISSION_FILE.backup.$(date +%Y%m%d_%H%M%S)"; echo "✅ Đã backup file cấu hình" ;;
        0) echo "Tạm biệt!"; exit 0 ;;
        *) echo -e "${RED}❌ Lựa chọn không hợp lệ!${NC}" ;;
    esac
    
    echo ""
    read -p "Nhấn Enter để tiếp tục..."
    clear
done
