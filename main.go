package main

import (
	"log/slog"
	"strconv"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	rl.InitWindow(960, 540, "Body Deformation")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.SetWindowState(rl.FlagWindowResizable)

	camera := rl.NewCamera3D(
		rl.NewVector3(10, 10, 10),
		rl.NewVector3(0, 0, 0),
		rl.NewVector3(0, 1, 0),
		45,
		rl.CameraPerspective,
	)
	cameraOrbiting := false
	// cameraOrbiting := true // TODO: Reset

	bodySize := rl.NewVector3(4, 4, 4)
	// bodySize := rl.NewVector3(4, 5, 3) // TODO: Reset
	bodySizesValue := [3]string{
		strconv.FormatFloat(float64(bodySize.X), 'f', 2, 64),
		strconv.FormatFloat(float64(bodySize.Y), 'f', 2, 64),
		strconv.FormatFloat(float64(bodySize.Z), 'f', 2, 64),
	}

	bodySplits := rl.NewVector3(2, 2, 2)
	// bodySplits := rl.NewVector3(4, 8, 3) // TODO: Reset
	bodySplitsValue := [3]string{
		strconv.FormatFloat(float64(bodySplits.X), 'f', 0, 64),
		strconv.FormatFloat(float64(bodySplits.Y), 'f', 0, 64),
		strconv.FormatFloat(float64(bodySplits.Z), 'f', 0, 64),
	}

	body := buildBodyShape(bodySize, bodySplits)

	var (
		bodySizesEdit  [3]bool
		bodySplitsEdit [3]bool
	)

	const (
		inputWidth    = 64
		inputHeight   = 24
		inputTextSize = 20
		padding       = 8
	)

	a0 := rl.NewVector3(0, 0, 0)
	aX := rl.NewVector3(1, 0, 0)
	aY := rl.NewVector3(0, 1, 0)
	aZ := rl.NewVector3(0, 0, 1)

	var numbers []rl.Texture2D
	for i := range 1000 {
		textImage := rl.ImageTextEx(rl.GetFontDefault(), strconv.Itoa(i), 32, 4, rl.White)
		numbers = append(numbers, rl.LoadTextureFromImage(textImage))
		rl.UnloadImage(textImage)
	}

	for !rl.WindowShouldClose() {
		uiRect := rl.NewRectangle(
			0, 0,
			padding+inputWidth*3+padding*2+padding,
			padding+inputHeight*2+padding+padding,
		)

		if rl.IsKeyPressed(rl.KeySpace) {
			cameraOrbiting = !cameraOrbiting
		}

		if rl.IsMouseButtonDown(rl.MouseButtonLeft) && !rl.CheckCollisionPointRec(rl.GetMousePosition(), uiRect) {
			md := rl.GetMouseDelta()
			rl.CameraYaw(&camera, -md.X*0.003, 1)
			rl.CameraPitch(&camera, -md.Y*0.003, 1, 1, 0)
		} else if cameraOrbiting {
			var rotation = rl.MatrixRotate(rl.GetCameraUp(&camera), 0.5*rl.GetFrameTime())
			var view = rl.Vector3Subtract(camera.Position, camera.Target)
			view = rl.Vector3Transform(view, rotation)
			camera.Position = rl.Vector3Add(camera.Target, view)
		}

		rl.CameraMoveToTarget(&camera, -rl.GetMouseWheelMove())
		if rl.IsKeyPressed(rl.KeyKpSubtract) {
			rl.CameraMoveToTarget(&camera, 2.0)
		}
		if rl.IsKeyPressed(rl.KeyKpAdd) {
			rl.CameraMoveToTarget(&camera, -2.0)
		}

		rl.BeginDrawing()
		{
			rl.ClearBackground(rl.RayWhite)

			rl.BeginMode3D(camera)
			{
				rl.DrawGrid(32, 1)

				// rl.DrawCube(rl.NewVector3(0, bodySize.Y/2, 0), bodySize.X, bodySize.Y, bodySize.Z, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(0, bodySize.Y/2, 0), bodySize.X, bodySize.Y, bodySize.Z, rl.Black)

				for i, p := range body {
					rl.DrawCube(p, 0.1, 0.1, 0.1, rl.Blue)
					rl.DrawBillboard(camera, numbers[i+1], rl.Vector3Add(p, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
				}

				rl.DrawLine3D(a0, aX, rl.Red)
				rl.DrawLine3D(a0, aY, rl.Green)
				rl.DrawLine3D(a0, aZ, rl.Blue)
			}
			rl.EndMode3D()

			rl.DrawRectangleRec(uiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(uiRect, 1, rl.Gray)

			bodyUpdated := false

			// Sizes
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*float32(i), padding, inputWidth, inputHeight),
					&bodySizesValue[i], inputTextSize, bodySizesEdit[i],
				) {
					bodySizesEdit[i] = !bodySizesEdit[i]

					v, err := strconv.ParseFloat(bodySizesValue[i], 32)
					if err != nil || v < 0 {
						slog.Error("Invalid size value", "err", err)
						var sv string
						switch i {
						case 0:
							sv = strconv.FormatFloat(float64(bodySize.X), 'f', 2, 64)
						case 1:
							sv = strconv.FormatFloat(float64(bodySize.Y), 'f', 2, 64)
						case 2:
							sv = strconv.FormatFloat(float64(bodySize.Z), 'f', 2, 64)
						}
						bodySizesValue[i] = sv
					} else {
						switch i {
						case 0:
							bodySize.X = float32(v)
						case 1:
							bodySize.Y = float32(v)
						case 2:
							bodySize.Z = float32(v)
						}
						bodyUpdated = true
					}
				}
			}

			// Splits
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*float32(i), padding+inputHeight+padding, inputWidth, inputHeight),
					&bodySplitsValue[i], inputTextSize, bodySplitsEdit[i],
				) {
					bodySplitsEdit[i] = !bodySplitsEdit[i]

					v, err := strconv.ParseInt(bodySplitsValue[i], 10, 32)
					if err != nil || v < 1 {
						slog.Error("Invalid split value", "err", err)
						var sv string
						switch i {
						case 0:
							sv = strconv.FormatInt(int64(bodySplits.X), 10)
						case 1:
							sv = strconv.FormatInt(int64(bodySplits.Y), 10)
						case 2:
							sv = strconv.FormatInt(int64(bodySplits.Z), 10)
						}
						bodySplitsValue[i] = sv
					} else {
						switch i {
						case 0:
							bodySplits.X = float32(v)
						case 1:
							bodySplits.Y = float32(v)
						case 2:
							bodySplits.Z = float32(v)
						}
						bodyUpdated = true
					}
				}
			}

			if bodyUpdated {
				body = buildBodyShape(bodySize, bodySplits)
			}
		}
		rl.EndDrawing()
	}
}

func buildBodyShape(size rl.Vector3, splits rl.Vector3) []rl.Vector3 {
	var body []rl.Vector3
	splits = rl.Vector3Scale(splits, 2)
	offset := rl.NewVector3(size.X/2, 0, size.Z/2)

	for y := range int(splits.Y) + 1 {
		for x := int(splits.X); x >= 0; x-- {
			for z := int(splits.Z); z >= 0; z-- {
				if (x%2 != 0 && z%2 != 0) || (x%2 != 0 && y%2 != 0) || (y%2 != 0 && z%2 != 0) {
					continue
				}

				body = append(body, rl.Vector3Subtract(rl.NewVector3(
					(float32(x)*size.X)/splits.X,
					(float32(y)*size.Y)/splits.Y,
					(float32(z)*size.Z)/splits.Z,
				), offset))
			}
		}
	}

	return body
}
