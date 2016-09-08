/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package model

import (
	"fmt"

	"github.com/unidoc/unidoc/common"
	. "github.com/unidoc/unidoc/pdf/core"
)

type PdfOutlineTreeNode struct {
	context interface{} // Allow accessing outer structure.
	First   *PdfOutlineTreeNode
	Last    *PdfOutlineTreeNode
}

// PDF outline dictionary (Table 152 - p. 376).
type PdfOutline struct {
	PdfOutlineTreeNode
	Count *int64
}

// Pdf outline item dictionary (Table 153 - pp. 376 - 377).
type PdfOutlineItem struct {
	PdfOutlineTreeNode
	Title  *PdfObjectString
	Parent *PdfOutlineTreeNode
	Prev   *PdfOutlineTreeNode
	Next   *PdfOutlineTreeNode
	Count  *int64
	Dest   PdfObject
	A      PdfObject
	SE     PdfObject
	C      PdfObject
	F      PdfObject
}

func NewPdfOutlineTree() *PdfOutline {
	outlineTree := PdfOutline{}
	outlineTree.context = &outlineTree
	return &outlineTree
}

func NewOutlineBookmark(title string, page *PdfIndirectObject) *PdfOutlineItem {
	bookmark := PdfOutlineItem{}
	bookmark.context = &bookmark

	bookmark.Title = MakeString(title)

	destArray := PdfObjectArray{}
	destArray = append(destArray, page)
	destArray = append(destArray, MakeName("Fit"))
	bookmark.Dest = &destArray

	return &bookmark
}

// Does not traverse the tree.
func newPdfOutlineFromDict(dict *PdfObjectDictionary) (*PdfOutline, error) {
	outline := PdfOutline{}
	outline.context = &outline

	if obj, hasType := (*dict)["Type"]; hasType {
		typeVal, ok := obj.(*PdfObjectName)
		if ok {
			if *typeVal != "Outlines" {
				common.Log.Error("Type != Outlines (%s)", *typeVal)
				// Should be "Outlines" if there, but some files have other types
				// Log as an error but do not quit.
				// Might be a good idea to log this kind of deviation from the standard separately.
			}
		}
	}

	if obj, hasCount := (*dict)["Count"]; hasCount {
		// This should always be an integer, but in a few cases has been a float.
		count, err := getNumberAsInt64(obj)
		if err != nil {
			return nil, err
		}
		outline.Count = &count
	}

	return &outline, nil
}

// Does not traverse the tree.
func (this *PdfReader) newPdfOutlineItemFromDict(dict *PdfObjectDictionary) (*PdfOutlineItem, error) {
	item := PdfOutlineItem{}
	item.context = &item

	// Title (required).
	obj, hasTitle := (*dict)["Title"]
	if !hasTitle {
		return nil, fmt.Errorf("Missing Title from Outline Item (required)")
	}
	obj, err := this.traceToObject(obj)
	if err != nil {
		return nil, err
	}
	title, ok := TraceToDirectObject(obj).(*PdfObjectString)
	if !ok {
		return nil, fmt.Errorf("Title not a string (%T)", obj)
	}
	item.Title = title

	// Count (optional).
	if obj, hasCount := (*dict)["Count"]; hasCount {
		countVal, ok := obj.(*PdfObjectInteger)
		if !ok {
			return nil, fmt.Errorf("Count not an integer (%T)", obj)
		}
		count := int64(*countVal)
		item.Count = &count
	}

	// Other keys.
	if obj, hasKey := (*dict)["Dest"]; hasKey {
		item.Dest, err = this.traceToObject(obj)
		if err != nil {
			return nil, err
		}
		err := this.traverseObjectData(item.Dest)
		if err != nil {
			return nil, err
		}
	}
	if obj, hasKey := (*dict)["A"]; hasKey {
		item.A, err = this.traceToObject(obj)
		if err != nil {
			return nil, err
		}
		err := this.traverseObjectData(item.A)
		if err != nil {
			return nil, err
		}
	}
	if obj, hasKey := (*dict)["SE"]; hasKey {
		item.SE, err = this.traceToObject(obj)
		if err != nil {
			return nil, err
		}
	}
	if obj, hasKey := (*dict)["C"]; hasKey {
		item.C, err = this.traceToObject(obj)
		if err != nil {
			return nil, err
		}
	}
	if obj, hasKey := (*dict)["F"]; hasKey {
		item.F, err = this.traceToObject(obj)
		if err != nil {
			return nil, err
		}
	}

	return &item, nil
}

// Get the outer object of the tree node (Outline or OutlineItem).
func (n *PdfOutlineTreeNode) getOuter() PdfObjectConvertible {
	if outline, isOutline := n.context.(*PdfOutline); isOutline {
		return outline
	}
	if outlineItem, isOutlineItem := n.context.(*PdfOutlineItem); isOutlineItem {
		return outlineItem
	}

	common.Log.Error("Invalid outline tree node item") // Should never happen.
	return nil
}

func (this *PdfOutlineTreeNode) GetContainingPdfObject() PdfObject {
	return this.getOuter().GetContainingPdfObject()
}

func (this *PdfOutlineTreeNode) ToPdfObject() PdfObject {
	return this.getOuter().ToPdfObject()
}

func (this *PdfOutline) GetContainingPdfObject() PdfObject {
	container := getCachedPdfObject(this)
	if container == nil {
		container := &PdfIndirectObject{}
		container.PdfObject = &PdfObjectDictionary{}
		cachePdfObjectConvertible(container, this)
		return container
	} else {
		return container
	}
}

// Recursively build the Outline tree PDF object.
func (this *PdfOutline) ToPdfObject() PdfObject {
	container := this.GetContainingPdfObject().(*PdfIndirectObject)
	dict := container.PdfObject.(*PdfObjectDictionary)

	(*dict)["Type"] = MakeName("Outlines")

	if this.First != nil {
		(*dict)["First"] = this.First.ToPdfObject()
	}

	if this.Last != nil {
		(*dict)["Last"] = this.Last.getOuter().GetContainingPdfObject()
		//PdfObjectConverterCache[this.Last.getOuter()]
	}

	return container
}

func (this *PdfOutlineItem) GetContainingPdfObject() PdfObject {
	container := getCachedPdfObject(this)
	if container == nil {
		container := &PdfIndirectObject{}
		container.PdfObject = &PdfObjectDictionary{}
		cachePdfObjectConvertible(container, this)
		return container
	} else {
		return container
	}
}

// Outline item.
// Recursively build the Outline tree PDF object.
func (this *PdfOutlineItem) ToPdfObject() PdfObject {
	container := this.GetContainingPdfObject().(*PdfIndirectObject)
	dict := container.PdfObject.(*PdfObjectDictionary)

	(*dict)["Title"] = this.Title
	if this.A != nil {
		(*dict)["A"] = this.A
	}
	if this.C != nil {
		(*dict)["C"] = this.C
	}
	if this.Dest != nil {
		(*dict)["Dest"] = this.Dest
	}
	if this.F != nil {
		(*dict)["F"] = this.F
	}
	if this.Count != nil {
		(*dict)["Count"] = MakeInteger(*this.Count)
	}
	if this.Next != nil {
		(*dict)["Next"] = this.Next.ToPdfObject()
	}
	if this.First != nil {
		(*dict)["First"] = this.First.ToPdfObject()
	}
	if this.Prev != nil {
		(*dict)["Prev"] = getCachedPdfObject(this.Prev.getOuter())
		//PdfObjectConverterCache[this.Prev.getOuter()]
	}
	if this.Last != nil {
		(*dict)["Last"] = getCachedPdfObject(this.Last.getOuter())
		// PdfObjectConverterCache[this.Last.getOuter()]
	}
	if this.Parent != nil {
		(*dict)["Parent"] = getCachedPdfObject(this.Parent.getOuter())
		//PdfObjectConverterCache[this.Parent.getOuter()]
	}

	return container
}