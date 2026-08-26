# Glossaire Français (French Glossary)

Ce document fournit des traductions standards françaises pour la terminologie clé du projet afin d'assurer la cohérence et la précision des traductions.

This document provides standard French translations for key project terminology to ensure consistency and accuracy in translations.

## Concepts de Base (Core Concepts)

- L'utilisation d'émojis dans les traductions est autorisée s'ils sont présents dans l'original
- L'utilisation de termes purement techniques est autorisée s'ils sont présents dans l'original
- L'utilisation de termes techniques en anglais est autorisée s'ils sont largement utilisés dans l'environnement technique francophone (par exemple, API)

| Chinois | Français | Anglais | Description |
|---------|----------|---------|-------------|
| 令牌 | Jeton | Token | Identifiants d'accès API ou unités de texte traitées par les modèles |
| 渠道 | Canal | Channel | Canal d'accès aux fournisseurs d'API |
| 分组 | Groupe | Group | Classification des utilisateurs ou des jetons |

## Modèles (Model Related)

| Chinois | Français | Anglais | Description |
|---------|----------|---------|-------------|
| 提示 | Invite | Prompt | Contenu d'entrée du modèle |
| 补全 | Complétion | Completion | Contenu de sortie du modèle. **Important :** Ne pas utiliser "Achèvement" ou "Finalisation" - uniquement "Complétion" pour correspondre à la terminologie technique |
| 输入 | Entrée | Input/Prompt | Contenu envoyé au modèle |
| 输出 | Sortie | Output/Completion | Contenu retourné par le modèle |

## Gestion des Utilisateurs (User Management)

| Chinois | Français | Anglais | Description |
|---------|----------|---------|-------------|
| 超级管理员 | Super-administrateur | Root User | Administrateur avec les privilèges les plus élevés |
| 管理员 | Administrateur | Admin User | Administrateur système |
| 普通用户 | Utilisateur normal | Normal User | Utilisateur avec privilèges standards |

## Gestion des Canaux (Channel Management)

| Chinois | Français | Anglais | Description |
|---------|----------|---------|-------------|
| 渠道 | Canal | Channel | Canal du fournisseur d'API |
| API密钥 | Clé API | API Key | Clé d'accès API. **Important :** Utiliser "Clé API" au lieu de "Jeton API" pour plus de précision et conformément à la terminologie technique francophone établie. Le terme "Clé" reflète mieux la fonctionnalité d'accès aux ressources, tandis que "Jeton" est plus souvent associé aux unités de texte dans le contexte du traitement des modèles linguistiques. |
| 优先级 | Priorité | Priority | Priorité de sélection du canal |
| 权重 | Poids | Weight | Poids d'équilibrage de charge |
| 代理 | Proxy | Proxy | Adresse du serveur proxy |
| 模型重定向 | Redirection de modèle | Model Mapping | Remplacement du nom du modèle dans le corps de la requête |
| 供应商 | Fournisseur | Provider/Vendor | Fournisseur de services ou d'API |

## Sécurité (Security Related)

| Chinois | Français | Anglais | Description |
|---------|----------|---------|-------------|
| 两步验证 | Authentification à deux facteurs | Two-Factor Authentication | Méthode de vérification de sécurité supplémentaire pour les comptes |
| 2FA | 2FA | Two-Factor Authentication | Abréviation de l'authentification à deux facteurs |

## Recommandations de Traduction (Translation Guidelines)

### Variantes Contextuelles de Traduction

**Invite/Entrée (Prompt/Input)**

- **Invite** : Lors de l'interaction avec les LLM, dans l'interface utilisateur, lors de la description de l'interaction avec le modèle
- **Entrée** : Dans la documentation technique et la description du processus de traitement des données
- **Règle** : S'il s'agit de l'expérience utilisateur et de l'interaction avec l'IA → "Invite", s'il s'agit du processus technique ou des calculs → "Entrée"

**Jeton (Token)**

- Jeton d'accès API (API Token)
- Unité de texte traitée par le modèle (Text Token)
- Jeton d'accès système (Access Token)

### Particularités de la Langue Française

- **Formes plurielles** : Nécessite une implémentation correcte des formes plurielles (_one, _other)
- **Accords grammaticaux** : Attention aux accords grammaticaux dans les termes techniques
- **Genre grammatical** : Accord du genre des termes techniques (par exemple, "modèle" - masculin, "canal" - masculin)

### Termes Standardisés

- **Complétion (Completion)** : Contenu de sortie du modèle
- **Fournisseur (Provider/Vendor)** : Organisation ou service fournissant des API ou des modèles d'IA

---

**Note pour les contributeurs :** Si vous trouvez des incohérences dans les traductions de terminologie ou si vous avez de meilleures suggestions de traduction pour le français, n'hésitez pas à créer une Issue ou une Pull Request.

**Contribution Note for French:** If you find any inconsistencies in terminology translations or have better translation suggestions for French, please feel free to submit an Issue or Pull Request.