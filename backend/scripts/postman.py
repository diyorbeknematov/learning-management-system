#!/usr/bin/env python3
"""Builds the Postman collection and environment from the Swagger file.

    make postman        (runs swagger first, then this script)

Input:  docs/swagger/swagger.json
Output: docs/postman/lms.postman_collection.json
        docs/postman/lms.postman_environment.json

Do not edit the output by hand; change the annotations of the handlers and run
make postman again.
"""
import json
import re
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SWAGGER = ROOT / "docs/swagger/swagger.json"
OUT = ROOT / "docs/postman"

# The order of the folders.
TAGS = ["Auth", "Users", "Uploads", "Categories", "Courses", "Modules", "Lessons",
        "Enrollments", "Progress", "Quizzes", "Questions", "Attempts",
        "Certificates", "Reviews", "Payments", "Finance", "Reports"]

# The auth folder is in the order a person uses it.
AUTH_ORDER = {p: i for i, p in enumerate(["/auth/register", "/auth/login", "/auth/refresh", "/auth/logout",
                                          "/auth/forgot-password", "/auth/reset-password"])}

# A path variable that is called {id} in the API gets a clearer name here.
ID_NAMES = {"/users/{id}": "userId", "/users/{id}/status": "userId",
            "/categories/{id}": "categoryId"}

# After a successful request the id of the new resource is kept in a variable,
# so the next requests of the folder work without copying ids by hand.
SAVE_ID = {
    ("post", "/users"): "userId",
    ("post", "/categories"): "categoryId",
    ("post", "/courses"): "courseId",
    ("post", "/courses/{courseId}/modules"): "moduleId",
    ("post", "/modules/{moduleId}/lessons"): "lessonId",
    ("post", "/lessons/{lessonId}/materials"): "materialId",
    ("post", "/courses/{courseId}/enrollments"): "enrollmentId",
    ("post", "/courses/{courseId}/quizzes"): "quizId",
    ("post", "/modules/{moduleId}/quizzes"): "quizId",
    ("post", "/quizzes/{quizId}/questions"): "questionId",
    ("post", "/quizzes/{quizId}/attempts"): "attemptId",
    ("post", "/courses/{courseId}/reviews"): "reviewId",
}

# Example values of body fields by name.
SAMPLES = {
    "email": "student@example.com",
    "username": "student1",
    "password": "Passw0rd!2024",
    "old_password": "Passw0rd!2024",
    "new_password": "N3w-Passw0rd!2024",
    "first_name": "Ali",
    "last_name": "Valiyev",
    "token": "{{reset_token}}",
    "refresh_token": "{{refresh_token}}",
}
ID_FIELDS = {"category_id": "{{categoryId}}", "instructor_id": "{{userId}}",
             "course_id": "{{courseId}}", "module_id": "{{moduleId}}",
             "role_id": "{{roleId}}"}


def resolve(schema, defs):
    """Follows $ref and allOf to the real schema."""
    if "$ref" in schema:
        return resolve(defs[schema["$ref"].split("/")[-1]], defs)
    if "allOf" in schema and len(schema["allOf"]) == 1:
        merged = dict(schema)
        merged.pop("allOf")
        merged.update(resolve(schema["allOf"][0], defs))
        return merged
    return schema


def example(schema, defs, name="", depth=0):
    schema = resolve(schema, defs)
    if depth > 6:
        return None
    if name in ID_FIELDS:
        return ID_FIELDS[name]
    if "enum" in schema:
        return schema["enum"][0]
    kind = schema.get("type")
    if kind == "object" or "properties" in schema:
        return {k: example(v, defs, k, depth + 1) for k, v in schema.get("properties", {}).items()}
    if kind == "array":
        return [example(schema.get("items", {}), defs, name, depth + 1)]
    if kind == "string":
        if name in SAMPLES:
            return SAMPLES[name]
        if name.endswith("_at") or name in ("from", "to", "date"):
            return "2026-01-01T00:00:00Z"
        if name.endswith("_url") or name == "url":
            return "https://example.com"
        return "string"
    if kind == "integer":
        return schema.get("minimum", 1) or 1
    if kind == "number":
        return schema.get("minimum", 0) or 0
    if kind == "boolean":
        return True
    return None


def path_variable(swagger_path, name):
    return ID_NAMES.get(swagger_path, name) if name == "id" else name


def expand_query(params, defs):
    """swag lists the fields of a query struct as separate query parameters."""
    return [p for p in params if p["in"] == "query"]


def save_script(lines):
    return {"listen": "test", "script": {"type": "text/javascript", "exec": lines}}


def build_request(path, method, op, defs):
    pvars = re.findall(r"\{(\w+)\}", path)
    url_path = path
    variables = []
    for name in pvars:
        pv = path_variable(path, name)
        url_path = url_path.replace("{" + name + "}", ":" + pv)
        variables.append({"key": pv, "value": "{{" + pv + "}}"})

    query = []
    for p in expand_query(op.get("parameters", []), defs):
        value = ""
        if "enum" in p:
            value = str(p["enum"][0])
        query.append({"key": p["name"], "value": value, "disabled": True,
                      "description": ", ".join(str(x) for x in p.get("enum", [])) or ""})

    raw = "{{base_url}}/api/v1" + url_path
    url = {"raw": raw, "host": ["{{base_url}}"],
           "path": ["api", "v1"] + [s for s in url_path.split("/") if s]}
    if variables:
        url["variable"] = variables
    if query:
        url["query"] = query

    request = {"method": method.upper(), "header": [], "url": url,
               "description": (op.get("summary", "") + "\n\n" + op.get("description", "")).strip()}

    body = next((p for p in op.get("parameters", []) if p["in"] == "body"), None)
    if body:
        request["header"].append({"key": "Content-Type", "value": "application/json"})
        request["body"] = {"mode": "raw", "options": {"raw": {"language": "json"}},
                           "raw": json.dumps(example(body["schema"], defs), indent=2, ensure_ascii=False)}

    if "security" not in op:
        request["auth"] = {"type": "noauth"}

    events = []
    if path in ("/auth/login", "/auth/register", "/auth/refresh"):
        events.append(save_script([
            "// keep the tokens for the next requests",
            "const body = pm.response.json();",
            "if (body.success) {",
            "  pm.environment.set('access_token', body.data.access_token);",
            "  pm.environment.set('refresh_token', body.data.refresh_token);",
            "}",
        ]))
    if path == "/auth/login":
        events[-1]["script"]["exec"].insert(3, "  if (body.data.user) { pm.environment.set('userId', body.data.user.id); }")
    if path == "/auth/logout":
        events.append(save_script(["if (pm.response.code === 204) { pm.environment.unset('refresh_token'); }"]))
    var = SAVE_ID.get((method, path))
    if var:
        events.append(save_script([
            "// keep the id of the new resource for the next requests",
            "const body = pm.response.json();",
            f"if (body.success && body.data && body.data.id) {{ pm.environment.set('{var}', body.data.id); }}",
        ]))

    item = {"name": op.get("summary", method.upper() + " " + path), "request": request}
    if events:
        item["event"] = events
    return item


def main():
    spec = json.loads(SWAGGER.read_text())
    defs = spec["definitions"]

    folders = {t: [] for t in TAGS}
    used = set(["base_url", "access_token", "refresh_token", "reset_token"])
    for path, methods in spec["paths"].items():
        for method, op in methods.items():
            tag = op["tags"][0]
            folders.setdefault(tag, [])
            item = build_request(path, method, op, defs)
            folders[tag].append((path, method, item))
            used.update(v["key"] for v in item["request"]["url"].get("variable", []))

    items = []
    for tag, entries in folders.items():
        if not entries:
            continue
        # a folder reads like a story: create first, then read, change, delete
        entries.sort(key=lambda e: (AUTH_ORDER.get(e[0], 99), ["post", "get", "put", "patch", "delete"].index(e[1]), e[0]))
        items.append({"name": tag, "item": [e[2] for e in entries]})

    collection = {
        "info": {
            "_postman_id": str(uuid.uuid5(uuid.NAMESPACE_URL, "lms-collection")),
            "name": spec["info"]["title"],
            "description": spec["info"]["description"] + "\n\nGenerated from docs/swagger/swagger.json by scripts/postman.py.",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
        },
        "auth": {"type": "bearer", "bearer": [{"key": "token", "value": "{{access_token}}", "type": "string"}]},
        "item": items,
    }

    defaults = {"base_url": "http://localhost:8080", "access_token": "", "refresh_token": "", "reset_token": ""}
    environment = {
        "id": str(uuid.uuid5(uuid.NAMESPACE_URL, "lms-environment")),
        "name": "LMS local",
        "values": [{"key": k, "value": defaults.get(k, ""), "type": "secret" if "token" in k else "default",
                    "enabled": True} for k in sorted(used, key=lambda k: (k not in defaults, k))],
        "_postman_variable_scope": "environment",
    }

    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "lms.postman_collection.json").write_text(json.dumps(collection, indent=2, ensure_ascii=False) + "\n")
    (OUT / "lms.postman_environment.json").write_text(json.dumps(environment, indent=2, ensure_ascii=False) + "\n")
    count = sum(len(f["item"]) for f in items)
    print(f"{count} requests in {len(items)} folders -> {OUT.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
