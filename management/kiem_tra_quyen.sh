#!/bin/bash

echo "🔍 KIỂM TRA QUYỀN BLOCKCHAIN"
echo "============================"

# Thông tin user hiện tại
echo "👤 User hiện tại: $(whoami)"
echo "📊 Các nhóm: $(groups)"

# Kiểm tra quyền file
echo ""
echo "📁 Quyền thư mục blockchain:"
ls -la ~/ethereum/node1/ | head -5

# Kiểm tra IPC socket
echo ""
echo "🔌 Quyền IPC socket:"
if [ -e ~/ethereum/node1/geth.ipc ]; then
    ls -la ~/ethereum/node1/geth.ipc
    echo "✅ IPC socket tồn tại"
else
    echo "❌ IPC socket không tồn tại (Geth chưa chạy?)"
fi

# Kiểm tra kết nối IPC
echo ""
echo "🔗 Test kết nối IPC:"
if ./build/bin/geth --exec "eth.blockNumber" attach ~/ethereum/node1/geth.ipc 2>/dev/null; then
    echo "✅ Có thể kết nối IPC"
else
    echo "❌ Không thể kết nối IPC"
fi

# Kiểm tra các API available
echo ""
echo "🔧 Các API có sẵn qua IPC:"
./build/bin/geth --exec "Object.keys(web3)" attach ~/ethereum/node1/geth.ipc 2>/dev/null

echo ""
echo "📋 TÓM TẮT QUYỀN:"
echo "=================="

# Kiểm tra từng loại quyền
if ./build/bin/geth --exec "typeof personal" attach ~/ethereum/node1/geth.ipc 2>/dev/null | grep -q "object"; then
    echo "✅ Personal API: Có quyền quản lý tài khoản"
else
    echo "❌ Personal API: Không có quyền"
fi

if ./build/bin/geth --exec "typeof miner" attach ~/ethereum/node1/geth.ipc 2>/dev/null | grep -q "object"; then
    echo "✅ Miner API: Có quyền điều khiển đào"
else
    echo "❌ Miner API: Không có quyền"
fi

if ./build/bin/geth --exec "typeof admin" attach ~/ethereum/node1/geth.ipc 2>/dev/null | grep -q "object"; then
    echo "✅ Admin API: Có quyền quản trị node"
else
    echo "❌ Admin API: Không có quyền"
fi

if ./build/bin/geth --exec "typeof security" attach ~/ethereum/node1/geth.ipc 2>/dev/null | grep -q "object"; then
    echo "✅ Security API: Có quyền quản lý bảo mật"
else
    echo "❌ Security API: Không có quyền"
fi

echo ""
echo "💡 Hướng dẫn:"
echo "- Nếu tất cả đều ✅: Bạn là Super Admin"
echo "- Nếu có một số ❌: Bạn có quyền hạn chế"
echo "- Nếu tất cả đều ❌: Bạn không có quyền admin"
