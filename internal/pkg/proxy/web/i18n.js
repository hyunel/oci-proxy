export const i18n = {
    en: {
        title: 'OCI Proxy',
        subtitle: 'Accelerate Docker image downloads through proxy, automatically generate download and rename commands',
        selectRuntime: 'Container Runtime',
        proxyAddress: 'Proxy Server Address',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: 'Enter your OCI Proxy server address',
        imageAddress: 'Image Address',
        imagePlaceholder: 'nginx:latest or docker.io/library/nginx:latest',
        imageExample: 'Supported formats: nginx:latest, ubuntu:22.04, registry.k8s.io/pause:3.9, etc.',
        copy: 'Copy',
        copied: 'Copied!',
        waitingInput: 'Please enter image address...',
        waitingProxy: 'Please enter proxy server address...',
        formatError: 'Invalid image address format'
    },
    zh: {
        title: 'OCI Proxy',
        subtitle: '通过代理加速下载 Docker 镜像，自动生成下载和重命名命令',
        selectRuntime: '容器运行时',
        proxyAddress: '代理服务器地址',
        proxyPlaceholder: 'proxy.example.com',
        proxyHint: '请输入您的 OCI Proxy 服务器地址',
        imageAddress: '镜像地址',
        imagePlaceholder: 'nginx:latest 或 docker.io/library/nginx:latest',
        imageExample: '支持格式: nginx:latest, ubuntu:22.04, registry.k8s.io/pause:3.9 等',
        copy: '复制',
        copied: '已复制!',
        waitingInput: '请输入镜像地址...',
        waitingProxy: '请输入代理服务器地址...',
        formatError: '镜像地址格式错误'
    }
};

export function detectLanguage() {
    const browserLang = navigator.language || navigator.userLanguage;
    return browserLang.toLowerCase().startsWith('zh') ? 'zh' : 'en';
}

export function translatePage(lang) {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (i18n[lang][key]) {
            el.textContent = i18n[lang][key];
        }
    });

    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        if (i18n[lang][key]) {
            el.placeholder = i18n[lang][key];
        }
    });

    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';
}
