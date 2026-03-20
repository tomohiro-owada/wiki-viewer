package main

import "html/template"

const commonCSS = `
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 16px;
  line-height: 1.6;
  color: #24292f;
  background: #f6f8fa;
}
a { color: #0969da; text-decoration: none; }
a:hover { text-decoration: underline; }
header {
  background: #24292f;
  color: #fff;
  padding: 12px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
}
header a { color: #fff; font-weight: 600; }
header .breadcrumb { font-size: 18px; font-weight: 400; opacity: 0.85; }
header .breadcrumb a { font-weight: 400; opacity: 0.85; }
header .breadcrumb span { opacity: 0.5; margin: 0 4px; }
header h1 { font-size: 18px; font-weight: 400; opacity: 0.85; }
.container {
  max-width: 1100px;
  margin: 24px auto;
  padding: 0 16px;
  display: flex;
  gap: 24px;
  align-items: flex-start;
}
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  padding: 16px;
  font-size: 14px;
  position: sticky;
  top: 24px;
}
.sidebar ul { list-style: none; padding-left: 0; }
.sidebar li { padding: 2px 0; }
.sidebar ul ul { padding-left: 16px; }
main {
  flex: 1;
  min-width: 0;
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  padding: 32px;
}
main h1 { font-size: 2em; margin-bottom: 16px; padding-bottom: 8px; border-bottom: 1px solid #d0d7de; }
main h2 { font-size: 1.5em; margin-top: 24px; margin-bottom: 12px; padding-bottom: 6px; border-bottom: 1px solid #eee; }
main h3 { font-size: 1.25em; margin-top: 20px; margin-bottom: 8px; }
main p { margin-bottom: 12px; }
main ul, main ol { margin-bottom: 12px; padding-left: 2em; }
main li { margin-bottom: 4px; }
main pre {
  background: #f6f8fa;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  padding: 16px;
  overflow-x: auto;
  margin-bottom: 16px;
  font-size: 14px;
}
main code {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 0.9em;
}
main :not(pre) > code {
  background: #eff1f3;
  padding: 2px 6px;
  border-radius: 4px;
}
main img { max-width: 100%; }
main blockquote {
  border-left: 4px solid #d0d7de;
  padding: 4px 16px;
  color: #57606a;
  margin-bottom: 12px;
}
main table {
  border-collapse: collapse;
  margin-bottom: 16px;
  width: 100%;
}
main th, main td {
  border: 1px solid #d0d7de;
  padding: 8px 12px;
  text-align: left;
}
main th { background: #f6f8fa; font-weight: 600; }
main .task-list-item { list-style: none; margin-left: -1.5em; }
main .task-list-item input { margin-right: 6px; }
footer.wiki-footer {
  max-width: 1100px;
  margin: 0 auto 24px;
  padding: 16px 16px 0;
  border-top: 1px solid #d0d7de;
  color: #57606a;
  font-size: 14px;
}
.not-found { text-align: center; padding: 48px 0; color: #57606a; }
.not-found h2 { border: none; color: #24292f; }
.page-list { margin-top: 24px; }
.page-list h3 { font-size: 16px; margin-bottom: 8px; }
.wiki-list { max-width: 700px; margin: 40px auto; padding: 0 16px; }
.wiki-list h1 { margin-bottom: 24px; }
.wiki-card {
  display: block;
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  padding: 16px 20px;
  margin-bottom: 12px;
  transition: border-color 0.15s;
}
.wiki-card:hover { border-color: #0969da; text-decoration: none; }
.wiki-card .name { font-size: 18px; font-weight: 600; }
.wiki-card .meta { font-size: 14px; color: #57606a; margin-top: 4px; }
@media (max-width: 768px) {
  .container { flex-direction: column; }
  .sidebar { width: 100%; position: static; }
}
`

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - Wiki</title>
<style>` + commonCSS + `</style>
</head>
<body>
<header>
  {{if .WikiName}}<a href="/">Wikis</a>
  <div class="breadcrumb"><a href="{{.HomeURL}}">{{.WikiName}}</a><span>/</span>{{.Title}}</div>
  {{else}}<a href="{{.HomeURL}}">Wiki</a>
  <h1>{{.Title}}</h1>
  {{end}}
</header>
<div class="container">
  {{if .HasSidebar}}<nav class="sidebar">{{.Sidebar}}</nav>{{end}}
  <main>{{.Body}}</main>
</div>
{{if .HasFooter}}<footer class="wiki-footer">{{.Footer}}</footer>{{end}}
<script type="module">
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
mermaid.initialize({startOnLoad:false,theme:'default'});
document.querySelectorAll('pre > code.language-mermaid').forEach((el)=>{
  const pre = el.parentElement;
  const div = document.createElement('div');
  div.className = 'mermaid';
  div.textContent = el.textContent;
  pre.replaceWith(div);
});
await mermaid.run({querySelector:'.mermaid'});
</script>
</body>
</html>
`))

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Wikis</title>
<style>` + commonCSS + `</style>
</head>
<body>
<header>
  <a href="/">Wikis</a>
</header>
<div class="wiki-list">
  <h1>Wikis</h1>
  {{range .}}<a class="wiki-card" href="{{.URL}}">
    <div class="name">{{.Display}}</div>
  </a>{{end}}
  {{if not .}}<p>No wikis found. Add subdirectories containing .md files.</p>{{end}}
</div>
</body>
</html>
`))
