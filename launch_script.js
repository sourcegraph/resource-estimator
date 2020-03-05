(async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const resp = await fetch(urlParams.get('dev') == 'true' ? 'http://localhost:8080/main.wasm' : `https://storage.googleapis.com/sourcegraph-resource-estimator/main_${document.currentScript.getAttribute('version')}.wasm`);
    if (!resp.ok) {
        const pre = document.createElement('pre');
        pre.innerText = await resp.text();
        document.body.appendChild(pre);
        return;
    }
    const src = await resp.arrayBuffer();
    const go = new Go();
    const result = await WebAssembly.instantiate(src, go.importObject);
    go.run(result.instance);
})();

