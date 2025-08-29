# AUR Package Migration Guide

## Objective
Simplify the AUR presence by consolidating from two packages to one:
- **Remove:** `godash` (source build, no calendar)  
- **Rename:** `godash-bin` → `godash` (pre-built with full features)

## Migration Steps

### 1. Delete the old `godash` package (source build)
```bash
# In your godash AUR repository clone
git rm PKGBUILD .SRCINFO
git commit -m "Remove source package - consolidating to single binary package"
git push
```

Then request deletion on AUR web interface or contact AUR admins.

### 2. Update `godash-bin` to become `godash`

In your `godash-bin` AUR repository:

```bash
# Replace files with new versions
cp PKGBUILD-new PKGBUILD
cp .SRCINFO-new .SRCINFO

# Update git
git add PKGBUILD .SRCINFO
git commit -m "Migrate godash-bin to godash - unified package with full features

- Simplified package name from godash-bin to godash
- Updated description to focus on features rather than build type  
- Maintained all functionality and dependencies
- Updated documentation and metadata"

git push
```

### 3. Request package name change
Contact AUR admins to rename the package from `godash-bin` to `godash`.

## Key Changes

### Package Metadata
- **Name:** `godash-bin` → `godash`
- **Description:** Updated to focus on features, not build method
- **Documentation:** Enhanced with comprehensive feature list
- **Removed:** Conflicts/provides entries (no longer needed)

### User Experience
- **Before:** `yay -S godash-bin` (confusing naming)
- **After:** `yay -S godash` (clean, simple)
- **Functionality:** Identical - full Google Calendar support

### Benefits
1. **Simplified choice:** One package, full features
2. **Better naming:** No "-bin" suffix confusion  
3. **Easier recommendation:** Just say "install godash"
4. **Cleaner documentation:** No need to explain multiple options

## Verification

After migration, users should be able to:
```bash
yay -S godash          # Install the full-featured version
godash                 # Run with all features including calendar
```

## Rollback Plan (if needed)
Keep backups of original PKGBUILD files and .SRCINFO in case rollback is needed.