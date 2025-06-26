---
title: Announcing Docula V2 - Living Documentation
slug: announcing-docula-v2
date: 2024-01-15
author: ZBZ Development Team
author_email: dev@zbz.dev
tags: [announcement, release, docula, v2]
excerpt: We're excited to announce Docula V2 with reactive updates, cloud storage, and beautiful templates!
featured_image: /images/docula-v2-hero.png
draft: false
---

# Announcing Docula V2 - Living Documentation

We're thrilled to announce the release of **Docula V2**, the most significant update to our documentation platform yet!

## 🚀 What's New

### ⚡ Reactive Updates
Your documentation now updates **automatically** when you change markdown files in cloud storage. No more manual builds or deployments!

### ☁️ Cloud-Native Storage
Store your documentation in **S3, GCS, MinIO**, or any Hodor-supported provider. Your docs live in the cloud and sync instantly.

### 🎨 Beautiful Templates
Choose from **multiple site templates**:
- **Documentation sites** for technical docs
- **Blog templates** for announcements  
- **Knowledge bases** for support content

### 🔧 Enhanced OpenAPI Integration
Your API documentation is now **automatically enhanced** with markdown content, creating rich, comprehensive specs.

## 💡 The Technology

Docula V2 is powered by our **Flux reactive system** and **Hodor cloud storage**:

```go
// Enable reactive updates with one line
contract := docula.DoculaContract{
    Storage: cloudStorage,  // Reactive updates enabled automatically!
}
```

## 🎯 Perfect for Teams

- **Developers** write docs in markdown alongside code
- **Technical writers** can edit content directly in cloud storage
- **Product managers** can update specs without deployments
- **Support teams** can maintain knowledge bases effortlessly

## 📈 What This Means

Docula V2 transforms documentation from a **static artifact** into a **living, breathing part** of your development workflow.

**Ready to upgrade?** Check out our [migration guide](/docs/migration) and experience the future of documentation!

---

*Happy documenting!*  
**The ZBZ Team** 🚀