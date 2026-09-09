export const i18n = {
    en: {
        subtitle: 'Type an image, copy the command.',
        selectRuntime: 'Container runtime',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: 'Proxy server address, click to edit',
        imageAddress: 'Image reference',
        imagePlaceholder: 'nginx:latest',
        imageExample: 'ubuntu:22.04 · bitnami/nginx · ghcr.io/astral-sh/uv · pasting a whole pull command works too · Enter to copy',
        generated: 'generated',
        copy: 'copy',
        copied: 'copied',
        waitingInput: 'Waiting for an image reference.',
        waitingProxy: 'Waiting for a proxy address.',
        statusPending: 'checking',
        statusUp: 'online',
        statusDown: 'unreachable'
    },
    zh: {
        subtitle: '填镜像名，复制命令。',
        selectRuntime: '容器运行时',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: '代理服务器地址，可点击修改',
        imageAddress: '镜像地址',
        imagePlaceholder: 'nginx:latest',
        imageExample: 'ubuntu:22.04 · bitnami/nginx · ghcr.io/astral-sh/uv · 也可粘贴完整 pull 命令 · 回车复制',
        generated: '生成结果',
        copy: '复制',
        copied: '已复制',
        waitingInput: '等待输入镜像地址。',
        waitingProxy: '等待输入代理地址。',
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
