// ==UserScript==
// @name         Hi音乐 FLAC下载管理助手 Pro
// @namespace    http://tampermonkey.net/
// @version      3.4-对接版
// @description  FLAC记录、批量下载、自定义文件名、对接远程下载器（优化版）
// @author       sxz799
// @match        https://flac.music.hi.cn/*
// @grant        GM_xmlhttpRequest
// @grant        GM_download
// @grant        GM_setValue
// @grant        GM_getValue
// @connect      other-lv.kuwo.cn
// @connect      kuwo.cn
// @connect      localhost
// @connect      127.0.0.1
// ==/UserScript==
(function () {
    "use strict";
    const STORAGE_KEY = "flac_download_records";
    const SERVER_CONFIG_KEY = "music_downloader_server";
    let downloading = false;
    let cancelDownload = false;
    const RETRY_TIMES = 2;
    
    // 获取服务器地址
    function getServerUrl() {
        return GM_getValue(SERVER_CONFIG_KEY, "http://localhost:8080");
    }
    
    // 设置服务器地址
    function setServerUrl(url) {
        GM_setValue(SERVER_CONFIG_KEY, url);
    }
    
    // 配置服务器地址
    function configServer() {
        const currentUrl = getServerUrl();
        const newUrl = prompt("请输入音乐下载器服务器地址：", currentUrl);
        if (newUrl !== null && newUrl.trim() !== "") {
            setServerUrl(newUrl.trim());
            alert("服务器地址已保存：" + newUrl.trim());
        }
    }
    
    // 推送单个音乐到下载器
    function pushToDownloader(item, showAlert = true) {
        const serverUrl = getServerUrl();
        return new Promise((resolve, reject) => {
            GM_xmlhttpRequest({
                method: "POST",
                url: `${serverUrl}/api/push`,
                headers: {
                    "Content-Type": "application/json"
                },
                data: JSON.stringify({
                    name: item.name,
                    url: item.url,
                    time: item.time
                }),
                onload(res) {
                    if (res.status >= 200 && res.status < 300) {
                        if (showAlert) alert(`已推送到下载器：${item.name}`);
                        resolve({ success: true });
                    } else {
                        if (showAlert) alert(`推送失败：${res.statusText}`);
                        reject(new Error(res.statusText));
                    }
                },
                onerror(err) {
                    if (showAlert) alert(`推送失败：${err.message}`);
                    reject(err);
                }
            });
        });
    }
    
    // 批量推送所有记录到下载器
    async function batchPushToDownloader() {
        const list = getRecords();
        if (!list.length) {
            alert("暂无记录可推送");
            return;
        }
        
        if (!confirm(`确定要推送 ${list.length} 首歌曲到下载器吗？`)) {
            return;
        }
        
        const serverUrl = getServerUrl();
        try {
            const response = await new Promise((resolve, reject) => {
                GM_xmlhttpRequest({
                    method: "POST",
                    url: `${serverUrl}/api/push/batch`,
                    headers: {
                        "Content-Type": "application/json"
                    },
                    data: JSON.stringify(list),
                    onload(res) {
                        if (res.status >= 200 && res.status < 300) {
                            resolve(res);
                        } else {
                            reject(new Error(res.statusText));
                        }
                    },
                    onerror(err) {
                        reject(err);
                    }
                });
            });
            
            alert(`成功推送 ${list.length} 首歌曲到下载器！`);
        } catch (err) {
            alert(`批量推送失败：${err.message}`);
        }
    }

    // 数据操作
    function getRecords() {
        try {
            return JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");
        } catch (e) {
            return [];
        }
    }

    function saveRecords(data) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
    }

    function clearRecords() {
        if (!confirm("确定清空所有记录吗？")) return;
        localStorage.removeItem(STORAGE_KEY);
        refreshPanel();
    }

    // 复制记录到剪切板
    async function exportRecords() {
        const list = getRecords();

        if (!list.length) {
            alert("暂无记录可复制");
            return;
        }

        // 转换为 JSON 格式
        const dataStr = JSON.stringify(list, null, 2);

        try {
            await navigator.clipboard.writeText(dataStr);
            alert(`已复制 ${list.length} 条记录到剪切板`);
        } catch (err) {
            console.error("复制失败:", err);

            // 兼容旧浏览器
            const textarea = document.createElement("textarea");
            textarea.value = dataStr;
            textarea.style.position = "fixed";
            textarea.style.opacity = "0";
            document.body.appendChild(textarea);
            textarea.select();

            try {
                document.execCommand("copy");
                alert(`已复制 ${list.length} 条记录到剪切板`);
            } catch (e) {
                alert("复制失败，请手动复制");
            }

            textarea.remove();
        }
    }

    // 添加记录
    function addRecord(name, url) {
        let list = getRecords();
        if (list.some(item => item.url === url)) return false;
        list.unshift({
            name: name,
            url: url,
            time: new Date().toLocaleString(),
            downloaded: false
        });
        saveRecords(list);
        refreshPanel();
        return true;
    }

    // 删除记录
    function deleteRecord(index) {
        let list = getRecords();
        list.splice(index, 1);
        saveRecords(list);
        refreshPanel();
    }

    // 文件名处理
    function safeName(name) {
        return name.replace(/[\\/:*?"<>|]/g, "_").trim();
    }

    // 获取真实文件后缀
    function getFileExtension(url, headers) {
        // 1. 从URL获取
        try {
            let pathname = new URL(url).pathname;
            let match = pathname.match(/\.([a-zA-Z0-9]+)$/);
            if (match) return "." + match[1].toLowerCase();
        } catch (e) { }

        // 2. 从Content-Type获取
        if (headers) {
            const contentType = headers.match(/content-type:\s*([^\s;]+)/i);
            if (contentType) {
                const typeMap = {
                    "audio/flac": ".flac",
                    "audio/mpeg": ".mp3",
                    "audio/wav": ".wav",
                    "audio/ogg": ".ogg",
                    "audio/aac": ".aac"
                };
                if (typeMap[contentType[1].toLowerCase()]) {
                    return typeMap[contentType[1].toLowerCase()];
                }
            }
        }

        // 3. 默认
        return ".flac";
    }

    // 单文件下载
    function downloadFile(item, index, retry = 0) {
        return new Promise(resolve => {
            GM_xmlhttpRequest({
                method: "GET",
                url: item.url,
                responseType: "blob",
                onload(res) {
                    const contentType = res.responseHeaders || "";
                    const ext = getFileExtension(item.url, contentType);
                    const blob = new Blob([res.response], { type: "audio/*" });
                    const blobUrl = URL.createObjectURL(blob);
                    const a = document.createElement("a");
                    a.href = blobUrl;
                    a.download = safeName(item.name) + ext;
                    document.body.appendChild(a);
                    a.click();
                    a.remove();
                    URL.revokeObjectURL(blobUrl);

                    const list = getRecords();
                    if (list[index]) {
                        list[index].downloaded = true;
                        saveRecords(list);
                    }
                    refreshPanel();
                    resolve({ success: true });
                },
                onerror() {
                    if (retry < RETRY_TIMES) {
                        console.log(`下载失败，重试 ${retry + 1}/${RETRY_TIMES}:`, item.name);
                        setTimeout(() => {
                            resolve(downloadFile(item, index, retry + 1));
                        }, 1000);
                    } else {
                        console.error("下载失败:", item.name);
                        resolve({ success: false, name: item.name });
                    }
                }
            });
        });
    }

    // 批量下载
    async function batchDownload() {
        if (downloading) {
            if (confirm("正在下载中，是否取消？")) {
                cancelDownload = true;
            }
            return;
        }

        let list = getRecords();
        if (!list.length) {
            alert("暂无记录");
            return;
        }

        downloading = true;
        cancelDownload = false;
        const failed = [];
        let btn = document.querySelector("#batch-download-btn");
        let cancelBtn = document.querySelector("#cancel-download-btn");

        for (let i = 0; i < list.length; i++) {
            if (cancelDownload) break;

            if (btn) {
                btn.innerHTML = `下载 ${i + 1}/${list.length}<br>${list[i].name}`;
            }
            if (cancelBtn) {
                cancelBtn.style.display = "inline-block";
            }

            const result = await downloadFile(list[i], i);
            if (!result.success) {
                failed.push(result.name);
            }

            if (i < list.length - 1 && !cancelDownload) {
                await new Promise(r => setTimeout(r, 1200));
            }
        }

        downloading = false;
        cancelDownload = false;
        refreshPanel();

        if (failed.length > 0) {
            alert(`批量下载完成，以下歌曲失败：\n${failed.join("\n")}`);
        } else {
            alert("批量下载完成");
        }
    }

    // 创建左下角面板
    function createPanel() {
        if (document.querySelector("#flac-record-panel")) return;
        const panel = document.createElement("div");
        panel.id = "flac-record-panel";
        panel.style.cssText = `
            position:fixed;
            left:10px;
            bottom:10px;
            width:380px;
            max-height:800px;
            overflow-y:auto;
            background:#fff;
            padding:12px;
            border-radius:12px;
            z-index:99999999;
            box-shadow:0 4px 20px rgba(0,0,0,.22);
            font-size:14px;
        `;
        document.body.appendChild(panel);
        refreshPanel();
    }

    // 刷新左下角列表
    function refreshPanel() {
        const panel = document.querySelector("#flac-record-panel");
        if (!panel) return;
        const list = getRecords();
        const serverUrl = getServerUrl();
        panel.innerHTML = `
            <div style="margin-bottom:12px;padding:8px;background:#f0f9ff;border-radius:8px;border:1px solid #91caff;font-size:12px;">
                <div style="font-weight:500;margin-bottom:4px;">🎯 远程下载器</div>
                <div style="color:#666;word-break:break-all;margin-bottom:6px;">${serverUrl}</div>
                <button id="config-server-btn" style="padding:4px 8px;border-radius:4px;border:1px solid #1677ff;background:#fff;color:#1677ff;cursor:pointer;font-size:12px;">
                    ⚙️ 配置
                </button>
            </div>
            <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:12px;">
                <div style="display:flex;flex-direction:column;gap:6px;">
                    <div style="display:flex;gap:6px;">
                       <div style="font-size:16px;font-weight:bold;">🎵 已记录 ${list.length}</div>
                    </div>
                    <div style="display:flex;gap:6px;">
                        <button id="batch-download-btn" class="ant-btn ant-btn-primary" style="height:30px;padding:0 10px;border-radius:6px;cursor:pointer;border:none;background:#1677ff;color:#fff;">
                            ⬇ 批量下载
                        </button>
                        <button id="batch-push-btn" style="height:30px;padding:0 10px;border-radius:6px;cursor:pointer;border:none;background:#722ed1;color:#fff;">
                            🚀 远程推送
                        </button>
                        <button id="cancel-download-btn" style="height:30px;padding:0 10px;border-radius:6px;cursor:pointer;border:1px solid #faad14;background:#fff;color:#faad14;display:none;">
                            ⏸ 取消
                        </button>
                        <button id="export-record-btn" style="height:30px;padding:0 10px;border-radius:6px;cursor:pointer;border:1px solid #52c41a;background:#fff;color:#52c41a;">
                            📤 导出
                        </button>
                        <button id="clear-record-btn" style="height:30px;padding:0 10px;border-radius:6px;cursor:pointer;border:1px solid #ff4d4f;background:#fff;color:#ff4d4f;">
                            🗑 清空
                        </button>
                    </div>
                </div>
            </div>
            ${list.length === 0 ? `
                <div style="text-align:center;color:#999;padding:20px 0;">暂无记录</div>
            ` : list.map((item, index) => `
                <div style="padding:10px;margin-bottom:8px;background:#fafafa;border-radius:8px;border:1px solid #eee;display:flex;align-items:center;justify-content:space-between;gap:10px;">
                    <div style="flex:1;min-width:0;">
                        <div title="${item.name}" style="font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">
                            ${item.name} ${item.downloaded ? "✅" : ""}
                        </div>
                        <div style="font-size:12px;color:#999;margin-top:4px;">${item.time}</div>
                    </div>
                    <div style="display:flex;gap:6px;flex-shrink:0;">
                        <button class="single-download" data-index="${index}" style="height:28px;padding:0 12px;border-radius:6px;border:none;background:#1677ff;color:#fff;cursor:pointer;">
                            ⬇
                        </button>
                        <button class="single-push" data-index="${index}" style="height:28px;padding:0 12px;border-radius:6px;border:none;background:#722ed1;color:#fff;cursor:pointer;">
                            🚀
                        </button>
                        <button class="delete-record" data-index="${index}" style="height:28px;padding:0 12px;border-radius:6px;border:1px solid #ff4d4f;background:#fff;color:#ff4d4f;cursor:pointer;">
                            🗑
                        </button>
                    </div>
                </div>
            `).join("")}
        `;

        // 绑定事件
        const configBtn = panel.querySelector("#config-server-btn");
        if (configBtn) configBtn.onclick = configServer;

        const batch = panel.querySelector("#batch-download-btn");
        if (batch) batch.onclick = batchDownload;

        const batchPushBtn = panel.querySelector("#batch-push-btn");
        if (batchPushBtn) batchPushBtn.onclick = batchPushToDownloader;

        const cancelBtn = panel.querySelector("#cancel-download-btn");
        if (cancelBtn) cancelBtn.onclick = () => { cancelDownload = true; };

        const clear = panel.querySelector("#clear-record-btn");
        if (clear) clear.onclick = clearRecords;

        const exportBtn = panel.querySelector("#export-record-btn");
        if (exportBtn) exportBtn.onclick = exportRecords;

        panel.querySelectorAll(".single-download").forEach(btn => {
            btn.onclick = function () {
                const index = Number(this.dataset.index);
                downloadFile(list[index], index);
            };
        });

        panel.querySelectorAll(".single-push").forEach(btn => {
            btn.onclick = function () {
                const index = Number(this.dataset.index);
                pushToDownloader(list[index]);
            };
        });

        panel.querySelectorAll(".delete-record").forEach(btn => {
            btn.onclick = function () {
                deleteRecord(Number(this.dataset.index));
            };
        });
    }

    // 添加记录按钮
    function addRecordButton() {
        const actions = document.querySelector(".download-actions");
        if (!actions) return;
        if (actions.querySelector(".flac-record-btn") && actions.querySelector(".flac-push-btn")) return;

        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "flac-record-btn";
        btn.innerHTML = "⭐ 记录";
        btn.style.cssText = `
            margin-left: 10px;
            padding: 8px 16px;
            border: none;
            border-radius: 6px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
        `;

        // 悬停效果
        btn.onmouseenter = function () {
            this.style.transform = "translateY(-2px)";
            this.style.boxShadow = "0 4px 12px rgba(102, 126, 234, 0.4)";
        };
        btn.onmouseleave = function () {
            this.style.transform = "translateY(0)";
            this.style.boxShadow = "0 2px 8px rgba(102, 126, 234, 0.3)";
        };

        btn.onclick = function () {
            const nameDom = document.querySelector(".download-detail .truncate");
            const name = nameDom ? nameDom.innerText.replace(/^名称[:：]\s*/, "").trim() : "未知歌曲";
            const link = actions.querySelector("a[href]");
            if (!link) {
                alert("没有找到下载地址");
                return;
            }
            if (addRecord(name, link.href)) {
                btn.innerHTML = "✓ 已记录";
                btn.style.background = "linear-gradient(135deg, #52c41a 0%, #389e0d 100%)";
            } else {
                btn.innerHTML = "记录已存在";
                btn.style.background = "linear-gradient(135deg, #c0b939ff 0%, #64671aff 100%)";
            }
            setTimeout(() => {
                btn.innerHTML = "⭐ 记录";
                btn.style.background = "linear-gradient(135deg, #667eea 0%, #764ba2 100%)";
            }, 1500);
        };
        actions.appendChild(btn);
        
        // 添加远程推送按钮
        const pushBtn = document.createElement("button");
        pushBtn.type = "button";
        pushBtn.className = "flac-push-btn";
        pushBtn.innerHTML = "🚀 远程推送";
        pushBtn.style.cssText = `
            margin-left: 10px;
            padding: 8px 16px;
            border: none;
            border-radius: 6px;
            background: linear-gradient(135deg, #722ed1 0%, #531dab 100%);
            color: white;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 2px 8px rgba(114, 46, 209, 0.3);
        `;

        // 悬停效果
        pushBtn.onmouseenter = function () {
            this.style.transform = "translateY(-2px)";
            this.style.boxShadow = "0 4px 12px rgba(114, 46, 209, 0.4)";
        };
        pushBtn.onmouseleave = function () {
            this.style.transform = "translateY(0)";
            this.style.boxShadow = "0 2px 8px rgba(114, 46, 209, 0.3)";
        };

        pushBtn.onclick = function () {
            const nameDom = document.querySelector(".download-detail .truncate");
            const name = nameDom ? nameDom.innerText.replace(/^名称[:：]\s*/, "").trim() : "未知歌曲";
            const link = actions.querySelector("a[href]");
            if (!link) {
                alert("没有找到下载地址");
                return;
            }
            
            pushToDownloader({
                name: name,
                url: link.href,
                time: new Date().toLocaleString()
            }).then(() => {
                pushBtn.innerHTML = "✓ 已远程推送";
                pushBtn.style.background = "linear-gradient(135deg, #52c41a 0%, #389e0d 100%)";
                setTimeout(() => {
                    pushBtn.innerHTML = "🚀 远程推送";
                    pushBtn.style.background = "linear-gradient(135deg, #722ed1 0%, #531dab 100%)";
                }, 1500);
            }).catch(() => {
                pushBtn.innerHTML = "✗ 远程推送失败";
                pushBtn.style.background = "linear-gradient(135deg, #ff4d4f 0%, #d9363e 100%)";
                setTimeout(() => {
                    pushBtn.innerHTML = "🚀 远程推送";
                    pushBtn.style.background = "linear-gradient(135deg, #722ed1 0%, #531dab 100%)";
                }, 1500);
            });
        };
        actions.appendChild(pushBtn);
    }

    // 页面监听
    const observer = new MutationObserver(() => {
        addRecordButton();
        if (!document.querySelector("#flac-record-panel")) {
            createPanel();
        }
    });
    observer.observe(document.body, { childList: true, subtree: true });

    // 初始化
    setTimeout(() => {
        createPanel();
        addRecordButton();
    }, 1000);
})();
