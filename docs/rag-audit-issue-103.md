# RAG 模块专项审计 (Issue #103)

审计日期: 2026-07-08 · 范围: `rag/` (约 10184 行非测试代码 / 31 文件)
方法: 逐个读取 retriever / store 变体的语义 + 全仓库交叉引用统计 (`grep` 构造函数引用, 分桶 examples / showcases / rag 内部 / 测试)。

> 结论先行: **不存在应立即删除的大块死代码**, 但有 1 个真正的零引用变体、1 处跨包重复实现、2 处可合并的薄包装。RAG 面向不同后端 (向量库 / 图库), 后端覆盖不可回退——本报告只出清单与迁移建议, 具体删除需另立 issue 逐项确认。

---

## 1. Retriever 功能矩阵 (`rag/retriever/`, 9 文件 / 2229 行)

统一实现 `rag.Retriever` (检索) 或 `rag.Reranker` (重排) 两个接口之一。

| 类型 (构造函数) | 行数 | 接口 | 策略 / 独有能力 | 外部依赖 |
|---|---|---|---|---|
| `VectorRetriever` (`NewVectorRetriever`) | 392 | Retriever | 向量相似度检索, 完整实现 (K/Config 变体) | 无 (依赖注入 VectorStore+Embedder) |
| `VectorStoreRetriever` (`NewVectorStoreRetriever`) | 同文件 | Retriever | **向后兼容薄包装**, 仅 topK 参数 | 同上 |
| `BM25Retriever` (`NewBM25Retriever` / `...WithTokenizer`) | 353 | Retriever | 稀疏关键词检索 (BM25 打分), 纯 Go | tokenizer 子包 |
| `GraphRetriever` (`NewGraphRetriever`) | 331 | Retriever | 基于知识图谱的检索 | KnowledgeGraph+Embedder |
| `HybridRetriever` (`NewHybridRetriever`) | 235 | Retriever | 加权融合多个 retriever (稠密+稀疏) | 组合其它 retriever |
| `SimpleReranker` (`NewSimpleReranker`) | 69 | Reranker | 关键词匹配重排, 纯 Go, 无 API | 无 |
| `LLMReranker` (`NewLLMReranker`) | 261 | Reranker | 用 LLM 打分重排 | llmtypes.Model |
| `CohereReranker` (`NewCohereReranker`) | 202 | Reranker | 调 Cohere rerank API | 外部 HTTP API + key |
| `JinaReranker` (`NewJinaReranker`) | 199 | Reranker | 调 Jina rerank API | 外部 HTTP API + key |
| `CrossEncoderReranker` (`NewCrossEncoderReranker`) | 187 | Reranker | 调 cross-encoder 模型服务重排 | 外部模型服务 |

**重叠点:**
- `VectorRetriever` ⊃ `VectorStoreRetriever` — 后者是前者的功能子集 (仅 topK)，自我标注 "backward compatibility"。
- 4 个远程 Reranker (`Cohere`/`Jina`/`CrossEncoder`) 仅差在打的是哪个 HTTP API + 请求/响应 schema; 结构高度同构 (~190–200 行/个)。`LLMReranker` 语义不同 (本地 LLM 打分), 不算重复。

**独有能力 (不可合并):** BM25 (稀疏), Graph (图谱), Hybrid (融合), LLMReranker (LLM 打分), SimpleReranker (纯 Go 无依赖)。

---

## 2. Store 功能矩阵 (`rag/store/`, 7 文件 / 2437 行)

| 类型 (构造函数) | 行数 | 接口 | 后端 | 外部依赖 |
|---|---|---|---|---|
| `InMemoryVectorStore` (`NewInMemoryVectorStore`) | 289 | VectorStore | 进程内内存 | 无 (纯 Go) |
| `ChromemVectorStore` (`NewChromemVectorStore` / `...Simple`) | 302 | VectorStore | chromem-go 嵌入式 (SQLite 持久化) | `philippgille/chromem-go` |
| `ChromaV2VectorStore` (`NewChromaV2VectorStore` / `...Simple`) | 639 | VectorStore | Chroma v2 HTTP 服务 | 网络 (Chroma server) |
| `FalkorDBGraph` (`NewFalkorDBGraph`) | 587 | KnowledgeGraph | FalkorDB 图库 (Redis 协议) | `redis` 客户端 |
| `falkordb_internal.go` (`NewGraph`) | 269 | (内部 Graph 助手) | FalkorDB 底层封装 | 同上 |
| `KnowledgeGraph` (`NewKnowledgeGraph`) | 222 | KnowledgeGraph | 通用图数据库 (databaseURL) | 驱动依赖 |
| `mock.go` (`NewMockEmbedder` / `NewSimpleReranker`) | 129 | Embedder / Reranker | 测试用 mock | 无 |

**后端覆盖图 (删减前必须保住):**
- 向量库: 内存 (InMemory) · 嵌入式 SQLite (Chromem) · 远程 HTTP (ChromaV2) — **三种不同部署形态, 均需保留**。
- 图库: FalkorDB (`FalkorDBGraph`) · 通用图 (`KnowledgeGraph`) — 两种不同后端。
- `falkordb_internal.go` 是 `falkordb.go` 的**底层助手**, 非独立 store, 不单独计数。
- `mock.go` 是测试脚手架, 保留。

**关键澄清:** `ChromaV2VectorStore` 与 `ChromemVectorStore` **不是重复** — 前者是 Chroma **v2 服务端 HTTP 客户端**, 后者是 `chromem-go` **嵌入式库**。命名相近但后端完全不同, 两者都要保留。

---

## 3. 交叉引用统计 (构造函数在自身文件/测试外的引用)

| 构造函数 | examples 文件数 | showcases | rag 内部文件 | 有测试 | 判定 |
|---|---|---|---|---|---|
| `NewInMemoryVectorStore` | 14 | 1 | 0 | ✅ | 重度使用 |
| `NewMockEmbedder` | 15 | 0 | 0 | ✅ | 重度使用 (测试基座) |
| `NewVectorStoreRetriever` | 12 | 0 | 1 | ✅ | 重度使用 (但见 §4 薄包装) |
| `NewFalkorDBGraph` | 4 | 0 | 0 | ✅ | 使用中 |
| `NewKnowledgeGraph` | 2 | 0 | 0 | ✅ | 使用中 |
| `NewBM25Retriever` (+WithTokenizer) | 2/1 | 0 | 1 | ✅ | 使用中 |
| `NewHybridRetriever` | 2 | 0 | 2 | ✅ | 使用中 |
| `NewVectorRetriever` | 1 | 0 | 2 | ✅ | 使用中 (内部为主) |
| `NewGraphRetriever` | 0 | 0 | 1 | ✅ | 仅内部 + 测试 |
| `NewLLMReranker` | 2 | 0 | 0 | ⚠️ 仅 example | 仅 example 演示 |
| `NewCohereReranker` | 1 | 0 | 0 | ⚠️ 仅 example | 仅 example 演示 |
| `NewJinaReranker` | 1 | 0 | 0 | ⚠️ 仅 example | 仅 example 演示 |
| `NewChromemVectorStore` (+Simple) | 1 | 0 | 0 | ✅ | 使用中 |
| `NewChromaV2VectorStore` (+Simple) | 1 | 0 | 0 | ❌ 无测试 | 使用中但无测试 |
| **`NewCrossEncoderReranker`** | **0** | **0** | **0** | **❌** | **零引用 (含字符串 "CrossEncoder" 全仓库无外部命中)** |

Reranker 通过 `rag.Reranker` 接口插入 pipeline (`pipeline.go:46,143` 的 `Reranker` 字段 + `rerankNode`), 因此内部引用=0 是正常的 (用户侧可插拔), 但**零 example + 零测试**的 `CrossEncoderReranker` 属于纯粹未被任何地方使用。

---

## 4. 可合并 / 可删除候选清单 + 迁移建议

按 "收益/风险比" 排序:

### 🔴 C1 — 删除 `CrossEncoderReranker` (零引用, 最高优先)
- **依据:** 全仓库对 `CrossEncoder` 无任何外部引用, 无 example, 无测试 (§3)。187 行纯死代码。
- **迁移:** 无调用方, 直接删除 `cross_encoder_reranker.go`。若想保留 "本地模型重排" 能力, 可在 doc 里指向 `LLMReranker`。
- **风险:** 极低。仅需确认没有下游仓库通过接口反射引用 (本模块内无)。

### 🟠 C2 — 消除跨包重复的 `SimpleReranker`
- **依据:** `rag/retriever/reranker.go:11` 与 `rag/store/mock.go:71` 各有一份 `SimpleReranker`/`NewSimpleReranker`, 实现几乎逐行相同 (仅一处 `float64(...)` 强转差异, 疑似复制后漂移)。两份都被 example 引用: `retriever.NewSimpleReranker()` (rag_reranker) 与 `store.NewSimpleReranker()` (rag_advanced)。
- **迁移:** 以 `rag/retriever` 版为准 (语义归属正确); 将 `store` 版标 `// Deprecated` 或改为转调 retriever 版; 更新 `examples/rag_advanced/main.go:125` 改用 `retriever.NewSimpleReranker()`; 移除 `store/mock.go` 中的重复后跑 `go test ./rag/...`。
- **风险:** 低 (非破坏性可先 deprecate)。

### 🟡 C3 — 收敛 `VectorStoreRetriever` 到 `VectorRetriever`
- **依据:** `VectorStoreRetriever` 自我标注 "backward compatibility", 是 `VectorRetriever` 的 topK-only 子集; 却被 examples 引用 12 次 (最广)。
- **迁移:** 不建议现在删 (引用面最大)。中期方案: 给 `VectorRetriever` 加一个 `NewVectorRetrieverTopK(store, embedder, topK)` 便捷构造, 将 `VectorStoreRetriever` 标 `// Deprecated` 指向它, 逐步迁移 12 处 example, 最后再删。
- **风险:** 中 (直接删会破坏大量 example)。必须走 deprecate → 迁移 → 删除三步。

### 🟢 C4 — (可选) 统一远程 Reranker 为配置驱动
- **依据:** `Cohere`/`Jina` (以及删掉的 CrossEncoder) 结构同构, 仅 API endpoint + 请求/响应 schema 不同, 各约 200 行。
- **迁移:** 可提取一个 `HTTPReranker` + provider 配置 (endpoint/headers/schema mapper), 三者收敛为一。但二者当前各有 example, 且外部 API 契约不同, 合并会引入 provider 抽象复杂度。
- **判定:** **暂不动** — 收益 (省约 200 行) 不抵抽象成本与破坏 example 的风险; 仅在未来新增第 3 个远程 provider 时再做。

### ✅ 明确保留 (不要动)
- 三个向量 store (InMemory / Chromem / ChromaV2) — 后端形态不同, 覆盖不可回退。
- 两个图 store (FalkorDB / KnowledgeGraph) + `falkordb_internal` 助手。
- BM25 / Graph / Hybrid / LLMReranker retriever — 各有独有语义。
- `mock.go` 的 `MockEmbedder` — 15 处测试/example 依赖。

---

## 5. 后续建议

1. **立即可做 (低风险):** C1 (删 CrossEncoder) + C2 (去重 SimpleReranker) —— 合计约 190+ 行, 非破坏性, 可一个 PR 完成。
2. **中期 (需迁移):** C3 走 deprecate 流程。
3. **补测试缺口:** `ChromaV2VectorStore` (639 行) 目前**无测试**, 建议补 mock HTTP 测试, 而非删除。
4. **不建议:** C4 的远程 reranker 合并 —— 除非新增 provider。

> 净精简空间: 短期约 190 行 (C1+C2) 可直接落地; 中期 C3 可再收敛约 40 行 + 降低 API 表面。这与 issue 预期的 "潜在精简空间最大" 相比偏保守 —— 原因是 10k 行里绝大部分是**真实、有区分度、有后端覆盖**的实现, 而非薄变体。
