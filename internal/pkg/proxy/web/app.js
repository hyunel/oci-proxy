import { i18n, detectLanguage, translatePage } from './i18n.js';

const currentLang = detectLanguage();
const t = key => i18n[currentLang][key];

let proxyInput, imageInput, commandText, copyBtn, copyLabel, output, promptRuntime;

function detectProxyAddress() {
    const { protocol, hostname, port } = window.location;
    const isDefaultPort = port === '' ||
        (protocol === 'http:' && port === '80') ||
        (protocol === 'https:' && port === '443');
    return isDefaultPort ? hostname : `${hostname}:${port}`;
}

function fitHost() {
    proxyInput.style.width = `${proxyInput.value.length || proxyInput.placeholder.length}ch`;
}

function normalizeImage() {
    const typed = imageInput.value.trim();
    const withoutCommand = typed.replace(/^\S*(?:docker|podman|nerdctl|ctr)\s+(?:images?\s+)?pull\s+/i, '');
    const host = proxyInput.value.trim();

    // pasting a reference that already carries the proxy host would double it up
    const image = host && withoutCommand.startsWith(`${host}/`)
        ? withoutCommand.slice(host.length + 1)
        : withoutCommand;

    if (image !== typed) {
        imageInput.value = image;
    }
}

function generateCommand() {
    const runtime = document.querySelector('input[name="runtime"]:checked').value;
    const host = proxyInput.value.trim();
    const image = imageInput.value.trim();

    promptRuntime.textContent = runtime;

    if (!host || !image) {
        commandText.textContent = t(host ? 'waitingInput' : 'waitingProxy');
        output.dataset.empty = 'true';
        copyBtn.disabled = true;
        return;
    }

    const proxied = `${host}/${image}`;
    commandText.textContent = [
        `${runtime} pull ${proxied}`,
        `${runtime} tag ${proxied} ${image}`,
        `${runtime} rmi ${proxied}`
    ].join(' && \\\n');
    output.dataset.empty = 'false';
    copyBtn.disabled = false;
}

async function copyToClipboard() {
    if (copyBtn.disabled) return;

    const text = commandText.textContent;
    try {
        await navigator.clipboard.writeText(text);
    } catch {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
    }

    copyBtn.classList.add('copied');
    copyLabel.textContent = t('copied');
    setTimeout(() => {
        copyBtn.classList.remove('copied');
        copyLabel.textContent = t('copy');
    }, 2000);
}

async function checkHealth() {
    const status = document.getElementById('status');
    const text = document.getElementById('status-text');
    try {
        const response = await fetch('/_/health');
        if (!response.ok) throw new Error(response.statusText);
        status.dataset.state = 'up';
        text.textContent = t('statusUp');
    } catch {
        status.dataset.state = 'down';
        text.textContent = t('statusDown');
    }
}

function init() {
    proxyInput = document.getElementById('proxy-address');
    imageInput = document.getElementById('image');
    commandText = document.getElementById('command-text');
    copyBtn = document.getElementById('copy-btn');
    copyLabel = document.getElementById('copy-label');
    output = document.getElementById('output');
    promptRuntime = document.getElementById('prompt-runtime');

    translatePage(currentLang);

    proxyInput.value = detectProxyAddress();
    fitHost();

    proxyInput.addEventListener('input', () => {
        fitHost();
        generateCommand();
    });
    imageInput.addEventListener('input', () => {
        normalizeImage();
        generateCommand();
    });
    imageInput.addEventListener('keydown', event => {
        if (event.key === 'Enter') {
            event.preventDefault();
            copyToClipboard();
        }
    });
    document.querySelectorAll('input[name="runtime"]').forEach(input => {
        input.addEventListener('change', generateCommand);
    });
    copyBtn.addEventListener('click', copyToClipboard);

    generateCommand();
    checkHealth();
    imageInput.focus();
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
