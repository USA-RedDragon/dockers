/*
 * Part of AREDN® -- Used for creating Amateur Radio Emergency Data Networks
 * Copyright (C) 2021-2025 Tim Wilkinson
 * Original Perl Copyright (C) 2015 Conrad Lara
 * See Contributors file for additional contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation version 3 of the License.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * Additional Terms:
 *
 * Additional use restrictions exist on the AREDN® trademark and logo.
 * See AREDNLicense.txt for more info.
 *
 * Attributions to the AREDN® Project must be retained in the source code.
 * If importing this code into a new or existing project attribution
 * to the AREDN® project must be added to the source code.
 *
 * You must not misrepresent the origin of the material contained within.
 *
 * Modified versions must be modified to attribute to the original source
 * and be marked in reasonable ways as differentiate it from the original
 * version
 */

import * as fs from "fs";

const pubsubbase = "/etc/meshlink";
const allpubsubbase = "/var/run/meshlink";

function add(pubsub, id, topic, data)
{
    if (id && topic && data) {
        const f = fs.open(`${pubsubbase}/${pubsub}`, "r+");
        if (f) {
            try {
                f.lock("x");
                const info = json(f.read("all") || '{"v1":[]}');
                for (let i = 0; i < length(info.v1); i++) {
                    if (info.v1[i].id == id) {
                        splice(info.v1, i, 1);
                        break;
                    }
                }
                push(info.v1, { id: id, topic: topic, data: data });
                f.seek();
                f.write(sprintf("%J", info));
                f.truncate(f.tell());
                f.lock("u");
                f.close();
                system(`echo "upload ${pubsub} ${pubsubbase}/${pubsub}" | socat -T 5 UNIX-CLIENT:/var/run/meshlink.sock - 2>&1 > /dev/null;`);
                return true;
            }
            catch (_) {
            }
            f.lock("u");
            f.close();
        }
    }
    return null;
}

function remove(pubsub, id)
{
    const f = fs.open(`${pubsubbase}/${pubsub}`, "r+");
    if (f) {
        try {
            f.lock("x");
            const info = json(f.read("all"));
            for (let i = 0; i < length(info.v1); i++) {
                if (info.v1[i].id == id) {
                    splice(info.v1, i, 1);
                    f.seek();
                    if (length(info.v1)) {
                        f.write(sprintf("%J", info));
                    }
                    f.truncate(f.tell());
                    f.lock("u");
                    f.close();
                    system(`echo "upload ${pubsub} ${pubsubbase}/${pubsub}" | socat -T 5 UNIX-CLIENT:/var/run/meshlink.sock - 2>&1 >/dev/null`);
                    return true;
                }
            }
        }
        catch (_) {
        }
        f.lock("u");
        f.close();
    }
    return false;
}

export function publish(id, topic, data)
{
    return add("publish", id, topic, data);
};

export function unpublish(id)
{
    return remove("publish", id);
};

function getByTopic(root, topic, targets)
{
    const results = [];
    const topicbase = substr(topic, -1) === "*" ? substr(topic, 0, -1) : null;
    const files = targets ? targets : fs.lsdir(root);
    if (files) {
        for (let i = 0; i < length(files); i++) {
            try {
                const file = `${root}/${files[i]}`;
                if (fs.lstat(file).size) {
                    const f = fs.open(file);
                    if (f) {
                        f.lock("s");
                        const filedata = f.read("all");
                        f.lock("u");
                        f.close();
                        const jl = json(filedata);
                        for (let k in jl) {
                            const j = jl[k];
                            for (let i = 0; i < length(j?.v1 ?? []); i++) {
                                const t = j.v1[i].topic;
                                if (t === topic || (topicbase && index(t, topicbase) === 0)) {
                                    push(results, j.v1[i].data);
                                }
                            }
                        }
                    }
                }
            }
            catch (_) {
            }
        }
    }
    return results;
}

export function published(topic, targets)
{
    return getByTopic(`${allpubsubbase}/publish`, topic, targets);
};

const watchers = {};

export function watch(type)
{
    const f = fs.popen(`echo $$;exec inotifywait --monitor --quiet --format '%w%f' --event close_write /var/run/meshlink/${type ?? "publish"}/`);
    watchers[f] = f.read("line");
    return f;
};

export function unwatch(handle)
{
    const pid = watchers[handle];
    if (pid) {
        system(`kill -9 ${pid}`);
        handle.close();
    }
};
