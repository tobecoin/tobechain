#!/bin/bash

case "$1" in
"super-admin")
    echo "👑 SUPER ADMIN CONSOLE"
    echo "====================="
    echo "Bạn có toàn quyền! Hãy cẩn thận."
    echo ""
    echo "Các lệnh có sẵn:"
    echo "1. personal.*     - Quản lý tài khoản"
    echo "2. miner.*        - Điều khiển đào"
    echo "3. admin.*        - Quản trị node"
    echo "4. security.*     - Quản lý bảo mật"
    echo "5. eth.*          - Blockchain operations"
    echo ""
    echo "Nhấn Ctrl+D để thoát console"
    echo ""
    ./build/bin/geth attach ~/ethereum/node1/geth.ipc
    ;;

"mining-admin")
    echo "⛏️ MINING ADMIN CONSOLE"
    echo "======================"
    echo "Bạn chỉ có quyền điều khiển đào coin."
    echo ""
    echo "Các lệnh được phép:"
    echo "- miner.start(1)                    // Bắt đầu đào"
    echo "- miner.stop()                      // Dừng đào"  
    echo "- miner.setEtherbase(\"0x...\")       // Đổi địa chỉ nhận thưởng"
    echo "- eth.mining                        // Kiểm tra trạng thái đào"
    echo "- eth.hashrate                      // Kiểm tra hashrate"
    echo "- eth.blockNumber                   // Xem block hiện tại"
    echo ""
    echo "KHÔNG được phép:"
    echo "- personal.* (quản lý tài khoản)"
    echo "- admin.* (quản trị node)"
    echo "- security.* (bảo mật)"
    echo ""
    ./build/bin/geth attach ~/ethereum/node1/geth.ipc
    ;;

"security-admin")
    echo "🛡️ SECURITY ADMIN CONSOLE"
    echo "========================="
    echo "Bạn chỉ có quyền quản lý bảo mật."
    echo ""
    echo "Các lệnh được phép:"
    echo "- security.getWhitelist()           // Xem danh sách trắng"
    echo "- security.getBlacklist()           // Xem danh sách đen"
    echo "- security.addToWhitelist(\"0x...\")  // Thêm vào whitelist"
    echo "- security.addToBlacklist(\"0x...\")  // Thêm vào blacklist"
    echo "- security.checkAddress(\"0x...\")    // Kiểm tra địa chỉ"
    echo "- eth.* (đọc blockchain)"
    echo ""
    echo "KHÔNG được phép:"
    echo "- personal.* (quản lý tài khoản)"
    echo "- miner.* (điều khiển đào)"
    echo "- admin.* (quản trị node)"
    echo ""
    ./build/bin/geth attach ~/ethereum/node1/geth.ipc
    ;;

"network-admin")
    echo "🌐 NETWORK ADMIN CONSOLE"
    echo "======================="
    echo "Bạn chỉ có quyền quản lý mạng."
    echo ""
    echo "Các lệnh được phép:"
    echo "- admin.peers                       // Xem các peers"
    echo "- admin.addPeer(\"enode://...\")      // Thêm peer"
    echo "- admin.removePeer(\"enode://...\")   // Xóa peer"
    echo "- admin.nodeInfo                    // Thông tin node"
    echo "- net.peerCount                     // Số lượng peers"
    echo "- net.listening                     // Trạng thái lắng nghe"
    echo ""
    echo "KHÔNG được phép:"
    echo "- personal.* (quản lý tài khoản)"
    echo "- miner.* (điều khiển đào)"
    echo "- security.* (bảo mật)"
    echo ""
    ./build/bin/geth attach ~/ethereum/node1/geth.ipc
    ;;

"read-only")
    echo "👀 READ ONLY CONSOLE"
    echo "==================="
    echo "Bạn chỉ có quyền đọc dữ liệu."
    echo ""
    echo "Các lệnh được phép:"
    echo "- eth.blockNumber                   // Số block hiện tại"
    echo "- eth.getBalance(\"0x...\")           // Xem số dư"
    echo "- eth.getBlock(number)              // Xem thông tin block"
    echo "- eth.getTransaction(\"0x...\")       // Xem giao dịch"
    echo "- eth.accounts                      // Xem tài khoản"
    echo "- net.version                       // Version mạng"
    echo ""
    echo "KHÔNG được phép:"
    echo "- Tất cả lệnh thay đổi dữ liệu"
    echo "- Không thể unlock, mining, admin"
    echo ""
    ./build/bin/geth attach ~/ethereum/node1/geth.ipc
    ;;

*)
    echo "🎭 BLOCKCHAIN ROLES MANAGER"
    echo "=========================="
    echo ""
    echo "Cách sử dụng: $0 [role]"
    echo ""
    echo "Các role có sẵn:"
    echo "  super-admin     👑 Toàn quyền (cẩn thận!)"
    echo "  mining-admin    ⛏️  Chỉ điều khiển đào"
    echo "  security-admin  🛡️  Chỉ quản lý bảo mật"
    echo "  network-admin   🌐 Chỉ quản lý mạng"
    echo "  read-only       👀 Chỉ đọc dữ liệu"
    echo ""
    echo "Ví dụ:"
    echo "  $0 super-admin      # Console với quyền tối cao"
    echo "  $0 mining-admin     # Console điều khiển đào"
    echo "  $0 security-admin   # Console quản lý bảo mật"
    echo ""
    echo "💡 Lưu ý: Script này chỉ hiển thị hướng dẫn."
    echo "   Việc thực thi quyền phụ thuộc vào cấu hình thực tế."
    ;;
esac
