import { i18n, detectLanguage, translatePage } from './i18n.js';

let currentLang = detectLanguage();

function generateCommand() {
    const runtime = document.querySelector('input[name="runtime"]:checked').value;
    const proxyAddress = document.getElementById('proxy-address').value.trim();
    const image = document.getElementById('image').value.trim();
    const commandText = document.getElementById('command-text');
    const copyBtn = document.getElementById('copy-btn');

    if (!proxyAddress) {
        commandText.textContent = i18n[currentLang].waitingProxy;
        copyBtn.disabled = true;
        return;
    }

    if (!image) {
        commandText.textContent = i18n[currentLang].waitingInput;
        copyBtn.disabled = true;
        return;
    }

    const proxyImage = `${proxyAddress}/${image}`;

    const pullCmd = `${runtime} pull ${proxyImage}`;
    const tagCmd = `${runtime} tag ${proxyImage} ${image}`;
    const rmiCmd = `${runtime} rmi ${proxyImage}`;

    const command = `${pullCmd} && \\\n${tagCmd} && \\\n${rmiCmd}`;

    commandText.textContent = command;
    copyBtn.disabled = false;
}

async function copyToClipboard() {
    const text = document.getElementById('command-text').textContent;
    const copyBtn = document.getElementById('copy-btn');

    if (copyBtn.disabled) return;

    try {
        await navigator.clipboard.writeText(text);
    } catch (err) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
    }

    const checkIcon = `<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>`;
    copyBtn.innerHTML = checkIcon;
    copyBtn.classList.add('copied');
    setTimeout(() => {
        const copyIcon = `<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>`;
        copyBtn.innerHTML = copyIcon;
        copyBtn.classList.remove('copied');
    }, 2000);
}

function init() {
    translatePage(currentLang);

    const port = window.location.port;
    const protocol = window.location.protocol;
    const hostname = window.location.hostname;
    const isDefaultPort = (protocol === 'http:' && port === '80') ||
                          (protocol === 'https:' && port === '443') ||
                          port === '';

    document.getElementById('proxy-address').value = isDefaultPort ? hostname : `${hostname}:${port}`;

    document.querySelectorAll('input[name="runtime"]').forEach(input => {
        input.addEventListener('change', generateCommand);
    });
    document.getElementById('proxy-address').addEventListener('input', generateCommand);
    document.getElementById('image').addEventListener('input', generateCommand);
    document.getElementById('copy-btn').addEventListener('click', copyToClipboard);

    generateCommand();
}if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
