
# 音乐下载器

一个基于 Golang + Vue 实现的音乐下载器程序。

## 功能特性

- 前端支持输入 JSON 数据
- 页面自动解析 JSON 并渲染成表格展示
- 预留代码块，方便后续补全 JSON 数据

## 项目结构

```
music-download/
├── backend/
│   ├── main.go
│   ├── go.mod
│   └── music.json
├── frontend/
│   ├── src/
│   │   ├── App.vue
│   │   └── main.js
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
└── README.md
```

## 安装与运行

### 后端（Golang）

1. 确保已安装 Go 1.21 或更高版本
2. 进入后端目录：
   ```bash
   cd backend
   ```
3. 运行后端服务：
   ```bash
   go run main.go
   ```
4. 后端服务将在 http://localhost:8080 启动

### 前端（Vue）

1. 确保已安装 Node.js 和 npm
2. 进入前端目录：
   ```bash
   cd frontend
   ```
3. 安装依赖：
   ```bash
   npm install
   ```
4. 启动开发服务器：
   ```bash
   npm run dev
   ```
5. 在浏览器中访问显示的地址

### 构建生产版本

```bash
cd frontend
npm run build
```

构建后的文件将放在 `frontend/dist` 目录中，可以直接通过后端服务访问。

## 使用说明

### 前端页面

1. 在代码块中输入你的 JSON 数据
2. 点击"解析并渲染"按钮
3. 页面会解析 JSON 并渲染成表格

### JSON 数据格式

```json
[
  {
    "name": "老街 - 李荣浩",
    "url": "https://other-er.kuwo.cn/9765575eced074d8b30c660334ca9bb1/6a55ddaa/resource/30106/trackmedia/F000004U927G0splK9.flac",
    "time": "2026/7/14 14:56:43",
    "downloaded": false
  },
  {
    "name": "模特 - 李荣浩",
    "url": "https://other-er.kuwo.cn/73ee5baabad446aec24c7dcb7641e222/6a55dda5/resource/1307392909/trackmedia/F000001nujtF0jebVT.flac",
    "time": "2026/7/14 14:56:37",
    "downloaded": false
  },
  {
    "name": "年少有为 - 李荣浩",
    "url": "https://other-lw.kuwo.cn/dddcf9255e1cdd9ba7d79420d82b4434/6a55dda0/resource/30106/trackmedia/F000003ZQREs39AMVp.flac",
    "time": "2026/7/14 14:56:32",
    "downloaded": false
  },
  {
    "name": "不将就-《何以笙箫默》电影片尾曲 - 李荣浩",
    "url": "https://other-lw.kuwo.cn/7547e3602a97459e475c226f09753af6/6a55dd98/resource/30106/trackmedia/F000001XmxKL31YXQm.flac",
    "time": "2026/7/14 14:56:24",
    "downloaded": false
  }
]


```

## 技术栈

- 后端：Golang
- 前端：Vue

## 开发

(待补充)

## 许可证

(待补充)

