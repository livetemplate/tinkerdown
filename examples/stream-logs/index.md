---
title: Stream Logs
---

# Stream Logs

Follows `demo-logs.sh` with a `stream` source. Run with `tinkerdown serve --allow-exec examples/stream-logs`.

## [All] | [Errors] level = error | [Warnings] level = warn

```lvt
<main lvt-source="all">
  <form name="Filter">
    <input type="search" name="f.msg" placeholder="search" lvt-on:input="Filter" lvt-mod:debounce="250">
  </form>
  {{if .Paused}}<button name="Resume">Resume</button>{{else}}<button name="Pause">Pause</button>{{end}}
  <button name="Clear">Clear</button>
  {{if .Error}}<p><mark>{{.Error}}</mark></p>{{end}}
  <ol class="log" lvt-fx:scroll="bottom-sticky">
    {{range .Data}}
    <li data-key="{{.seq}}" class="lvl-{{.level}} stream-{{.stream}}"><time>{{.time}}</time> <b>{{.level}}</b> {{.msg}}</li>
    {{end}}
  </ol>
</main>
```

<style>
.log { max-height: 60vh; overflow-y: auto; font-family: monospace; list-style: none; padding: 0; }
.log .lvl-error, .log .lvl-fatal { color: #c0392b; }
.log .lvl-warn { color: #b9770e; }
.log .stream-system { opacity: .6; font-style: italic; }
</style>
