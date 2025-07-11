package handlers


type ViewBook struct {
    ID              int32    // Only for delete
    Title           string
    Authors         []string
    Subjects        []string
    PublishYear     int
    Description     string
    CoverArtURL     string
    WorkID          string
    ShowDeleteButton bool   // 👈 control logic
}