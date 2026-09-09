import { i18n, detectLanguage, translatePage } from './i18n.js';

const currentLang = detectLanguage();
const t = key => i18n[currentLang][key];

const DOCKER_HUB = 'docker.io';

let mode = 'command';
let proxyInput, imageInput, commandText, copyBtn, copyLabel;
let output, outputPath, outputNote, promptRuntime, modeButtons;

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

// same rule as the docker CLI: only a domain may hold ".", ":" or uppercase
function registryOf(image) {
    const slash = image.indexOf('/');
    if (slash < 0) {
        return DOCKER_HUB;
    }
    const first = image.slice(0, slash);
    const isDomain = /[.:]/.test(first) || first === 'localhost' || first !== first.toLowerCase();
    return isDomain ? first : DOCKER_HUB;
}

function commandBlock(runtime, host, image) {
    if (!host || !image) {
        return { body: t(host ? 'waitingInput' : 'waitingProxy'), empty: true };
    }

    const proxied = `${host}/${image}`;
    return {
        body: [
            `${runtime} pull ${proxied}`,
            `${runtime} tag ${proxied} ${image}`,
            `${runtime} rmi ${proxied}`
        ].join(' && \\\n')
    };
}

function mirrorBlock(runtime, host, image) {
    if (!host) {
        return { body: t('waitingProxy'), empty: true };
    }

    const registry = registryOf(image);
    const isHub = registry === DOCKER_HUB;
    const pull = `${runtime} pull ${image || 'nginx:latest'}`;

    if (runtime === 'docker') {
        if (!isHub) {
            return {
                body: t('dockerHubOnlyBody').replace('{registry}', registry),
                note: t('dockerHubOnlyNote').replace('{registry}', registry),
                warn: true,
                empty: true
            };
        }
        return {
            path: '/etc/docker/daemon.json',
            body: `{\n  "registry-mirrors": ["https://${host}"]\n}`,
            note: `${t('restartDocker')} ${t('thenPull')} ${pull}`
        };
    }

    if (runtime === 'podman') {
        return {
            path: '/etc/containers/registries.conf',
            body: [
                '[[registry]]',
                `prefix = "${registry}"`,
                `location = "${isHub ? host : `${host}/${registry}`}"`
            ].join('\n'),
            note: `${t('podmanApplies')} ${t('thenPull')} ${pull}`
        };
    }

    return {
        path: `/etc/containerd/certs.d/${registry}/hosts.toml`,
        body: [
            `server = "https://${isHub ? 'registry-1.docker.io' : registry}"`,
            '',
            `[host."https://${host}/v2${isHub ? '' : `/${registry}`}"]`,
            '  capabilities = ["pull", "resolve"]',
            '  override_path = true'
        ].join('\n'),
        note: `${t('restartContainerd')} ${t('thenPull')} ${pull}`
    };
}

function render() {
    const runtime = document.querySelector('input[name="runtime"]:checked').value;
    const host = proxyInput.value.trim();
    const image = imageInput.value.trim();

    promptRuntime.textContent = runtime;

    const block = mode === 'mirror'
        ? mirrorBlock(runtime, host, image)
        : commandBlock(runtime, host, image);

    commandText.textContent = block.body;
    output.dataset.empty = block.empty ? 'true' : 'false';
    copyBtn.disabled = Boolean(block.empty);

    outputPath.textContent = block.path || '';
    outputPath.hidden = !block.path;
    outputNote.textContent = block.note || '';
    outputNote.hidden = !block.note;
    outputNote.dataset.warn = block.warn ? 'true' : 'false';
}

function setMode(next) {
    mode = next;
    modeButtons.forEach(button => {
        button.setAttribute('aria-selected', String(button.dataset.mode === next));
    });
    render();
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
    outputPath = document.getElementById('output-path');
    outputNote = document.getElementById('output-note');
    promptRuntime = document.getElementById('prompt-runtime');
    modeButtons = [...document.querySelectorAll('.mode')];

    translatePage(currentLang);

    proxyInput.value = detectProxyAddress();
    fitHost();

    proxyInput.addEventListener('input', () => {
        fitHost();
        render();
    });
    imageInput.addEventListener('input', () => {
        normalizeImage();
        render();
    });
    imageInput.addEventListener('keydown', event => {
        if (event.key === 'Enter') {
            event.preventDefault();
            copyToClipboard();
        }
    });
    document.querySelectorAll('input[name="runtime"]').forEach(input => {
        input.addEventListener('change', render);
    });
    modeButtons.forEach(button => {
        button.addEventListener('click', () => setMode(button.dataset.mode));
    });
    copyBtn.addEventListener('click', copyToClipboard);

    render();
    checkHealth();
    imageInput.focus();
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
