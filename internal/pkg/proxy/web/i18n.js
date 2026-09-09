export const i18n = {
    en: {
        subtitle: 'Pull any image through the proxy, once or for good.',
        selectRuntime: 'Container runtime',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: 'Proxy server address, click to edit',
        imageAddress: 'Image reference',
        imagePlaceholder: 'nginx:latest',
        imageExample: 'ubuntu:22.04 · bitnami/nginx · ghcr.io/astral-sh/uv · pasting a whole pull command works too · Enter to copy',
        modeCommand: 'command',
        modeMirror: 'mirror config',
        copy: 'copy',
        copied: 'copied',
        waitingInput: 'Waiting for an image reference.',
        waitingProxy: 'Waiting for a proxy address.',
        thenPull: 'After that, pull normally:',
        restartDocker: 'Merge into the file, then restart Docker.',
        restartContainerd: 'Write the file, then restart containerd.',
        podmanApplies: 'Append to the file; it applies immediately.',
        dockerHubOnlyBody: 'Docker only applies registry-mirrors to Docker Hub, so {registry} cannot be mirrored this way.',
        dockerHubOnlyNote: 'Use the command tab for {registry}, or switch the host to containerd or Podman, which can mirror any registry.',
        statusPending: 'checking',
        statusUp: 'online',
        statusDown: 'unreachable'
    },
    zh: {
        subtitle: '通过代理拉取任意镜像，临时一次或一劳永逸。',
        selectRuntime: '容器运行时',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: '代理服务器地址，可点击修改',
        imageAddress: '镜像地址',
        imagePlaceholder: 'nginx:latest',
        imageExample: 'ubuntu:22.04 · bitnami/nginx · ghcr.io/astral-sh/uv · 也可粘贴完整 pull 命令 · 回车复制',
        modeCommand: '单次命令',
        modeMirror: '镜像源配置',
        copy: '复制',
        copied: '已复制',
        waitingInput: '等待输入镜像地址。',
        waitingProxy: '等待输入代理地址。',
        thenPull: '此后直接拉取即可：',
        restartDocker: '合并进该文件，然后重启 Docker。',
        restartContainerd: '写入该文件，然后重启 containerd。',
        podmanApplies: '追加到该文件，立即生效。',
        dockerHubOnlyBody: 'Docker 的 registry-mirrors 只对 Docker Hub 生效，{registry} 无法用这种方式加速。',
        dockerHubOnlyNote: '{registry} 请改用「单次命令」，或把宿主换成 containerd / Podman——它们支持为任意 registry 配置镜像源。',
        statusPending: '检测中',
        statusUp: '在线',
        statusDown: '无法连接'
    }
};

export function detectLanguage() {
    const browserLang = navigator.language || navigator.userLanguage;
    return browserLang.toLowerCase().startsWith('zh') ? 'zh' : 'en';
}

export function translatePage(lang) {
    const apply = (attribute, set) => {
        document.querySelectorAll(`[${attribute}]`).forEach(el => {
            const value = i18n[lang][el.getAttribute(attribute)];
            if (value) {
                set(el, value);
            }
        });
    };

    apply('data-i18n', (el, value) => { el.textContent = value; });
    apply('data-i18n-placeholder', (el, value) => { el.placeholder = value; });
    apply('data-i18n-title', (el, value) => { el.title = value; });
    apply('data-i18n-aria-label', (el, value) => { el.setAttribute('aria-label', value); });

    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';
}
